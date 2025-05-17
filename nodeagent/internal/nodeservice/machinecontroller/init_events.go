package machinecontroller

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"google.golang.org/protobuf/types/known/emptypb"
	"log/slog"
	"time"
)

var StatesToListenInit = []types.MachineState{
	types.MachineStateStarting,
	types.MachineStateRunning,
	types.MachineStateDegraded,
}

func (c *Controller) syncInitEventsListener() {
	c.initListenerMutex.Lock()
	defer c.initListenerMutex.Unlock()

	machine := c.GetMachine()
	c.log.Debug("syncing init listener", slog.Any("state", machine.State))
	if !typeutil.Includes(StatesToListenInit, machine.State) {
		if c.cancelInitListener != nil {
			c.cancelInitListener()
			c.cancelInitListener = nil
		}
		return
	} else if c.cancelInitListener != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelInitListener = cancel
	go c.listenToInitEvents(ctx, machine.ID)
}

func (c *Controller) listenToInitEvents(ctx context.Context, machineID string) {
	consecutiveErrorCount := 0
	ticker := time.NewTicker(1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.handleInitEventStream(ctx, machineID, &consecutiveErrorCount); err != nil {
				c.log.Debug("init event stream handler failed",
					slog.Any("error", err),
					slog.Int("consecutive-error", consecutiveErrorCount))
				if consecutiveErrorCount >= 3 {
					c.dispatchMachineStateChangeEvent(types.MachineStateError)
				} else if consecutiveErrorCount > 0 {
					c.dispatchMachineStateChangeEvent(types.MachineStateDegraded)
				}
				ticker.Reset(time.Second)
			}
		}
	}
}

func (c *Controller) handleInitEventStream(ctx context.Context, machineID string, consecutiveErrorCount *int) error {
	client, closeClient := c.runtimeProvider.NewInitClient(machineID)
	defer closeClient()

	stream, err := client.Events(ctx, connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		*consecutiveErrorCount++
		return fmt.Errorf("failed to connect to init daemon: %w", err)
	}

	hasReceived := false
	for stream.Receive() {
		if !hasReceived {
			if *consecutiveErrorCount > 0 {
				c.dispatchMachineStateChangeEvent(types.MachineStateRunning)
				*consecutiveErrorCount = 0
			}
			hasReceived = true
		}

		msg := stream.Msg()
		c.log.Debug("received event from init", slog.Any("event", msg))
		//c.eventBus.PublishEvent(stream.Msg())
	}

	err = stream.Err()
	if !hasReceived && err != nil {
		*consecutiveErrorCount++
	}

	return err
}
