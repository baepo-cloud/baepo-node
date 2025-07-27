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

type RuntimeListener struct {
	Cancel                context.CancelFunc
	ConsecutiveErrorCount int
}

func (c *Controller) shouldStartRuntimeListener(machine *types.Machine) bool {
	return typeutil.Includes([]coretypes.MachineState{
		coretypes.MachineStateRunning,
		coretypes.MachineStateDegraded,
	}, machine.State)
}

func (c *Controller) startRuntimeListener(machine *types.Machine) {
	c.log.Debug("starting runtime listener")
	ctx, cancel := context.WithCancel(context.Background())
	_ = c.SetState(func(state *State) error {
		state.RuntimeListener = &RuntimeListener{
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
					state.RuntimeListener = nil
					return nil
				})
				return
			case <-ticker.C:
				err := c.connectToRuntimeListener(ctx, machine.ID)
				c.eventBus.PublishEvent(&RuntimeListenerDisconnectedMessage{Error: err})
				c.log.Debug("runtime listener disconnected", slog.Any("error", err))
				if err != nil {
					ticker.Reset(time.Second)
				}
			}
		}
	}()
}

func (c *Controller) connectToRuntimeListener(ctx context.Context, machineID string) error {
	client, closeClient := c.runtimeService.GetClient(machineID)
	defer closeClient()

	stream, err := client.Events(ctx, connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		return err
	}

	hasReceived := false
	for stream.Receive() {
		if !hasReceived {
			c.eventBus.PublishEvent(&RuntimeListenerConnectedMessage{})
			hasReceived = true
		}

		msg := stream.Msg()
		if event, ok := msg.Event.(*nodev1pb.RuntimeEventsResponse_ContainerStateChanged); ok {
			c.eventBus.PublishEvent(&ContainerStateChangedMessage{
				EventID:   msg.EventId,
				Event:     event.ContainerStateChanged,
				Timestamp: msg.Timestamp.AsTime(),
			})
		}
	}
	return stream.Err()
}
