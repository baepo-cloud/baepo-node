package machinecontroller

import (
	"connectrpc.com/connect"
	"context"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"google.golang.org/protobuf/types/known/emptypb"
	"log/slog"
	"time"
)

type InitListener struct {
	Cancel                context.CancelFunc
	ConsecutiveErrorCount int
}

func (c *Controller) shouldStartInitListener(machine *types.Machine) bool {
	return typeutil.Includes([]coretypes.MachineState{
		coretypes.MachineStateRunning,
		coretypes.MachineStateDegraded,
	}, machine.State)
}

func (c *Controller) startInitListener(machine *types.Machine) {
	c.log.Debug("start init listener")
	ctx, cancel := context.WithCancel(context.Background())
	_ = c.SetState(func(state *State) error {
		state.InitListener = &InitListener{
			Cancel:                cancel,
			ConsecutiveErrorCount: 0,
		}
		return nil
	})

	c.wg.Add(1)
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		defer c.wg.Done()

		for {
			select {
			case <-ctx.Done():
				_ = c.SetState(func(state *State) error {
					state.InitListener = nil
					return nil
				})
				return
			case <-ticker.C:
				err := c.connectToInitListener(ctx, machine.ID)
				c.eventBus.PublishEvent(&InitListenerDisconnectedMessage{Error: err})
				c.log.Debug("init listener disconnected", slog.Any("error", err))
				if err != nil {
					ticker.Reset(time.Second)
				}
			}
		}
	}()
}

func (c *Controller) connectToInitListener(ctx context.Context, machineID string) error {
	client, closeClient := c.runtimeProvider.NewInitClient(machineID)
	defer closeClient()

	stream, err := client.Events(ctx, connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		return err
	}

	hasReceived := false
	for stream.Receive() {
		if !hasReceived {
			c.eventBus.PublishEvent(&InitListenerConnectedMessage{})
			hasReceived = true
		}

		msg := stream.Msg()
		if event, ok := msg.Event.(*nodev1pb.InitEventsResponse_ContainerStateChanged); ok {
			c.eventBus.PublishEvent(&InitContainerStateChangedMessage{
				EventID:   msg.EventId,
				Event:     event.ContainerStateChanged,
				Timestamp: msg.Timestamp.AsTime(),
			})
		}
	}
	return stream.Err()
}
