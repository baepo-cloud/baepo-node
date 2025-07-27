package machinecontroller

import (
	"context"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"time"
)

type (
	AssessStateMessage struct{}

	DesiredStateChangedMessage struct {
		DesiredState coretypes.MachineDesiredState
		Timestamp    time.Time
	}

	StateChangedMessage struct {
		State     coretypes.MachineState
		Timestamp time.Time
	}

	ReconciliationCompleteMessage struct {
		Success bool
		Error   error
	}

	ContainerStateChangedMessage struct {
		EventID   string
		Event     *nodev1pb.RuntimeEventsResponse_ContainerStateChangedEvent
		Timestamp time.Time
	}

	RuntimeListenerConnectedMessage struct{}

	RuntimeListenerDisconnectedMessage struct {
		Error error
	}
)

func (c *Controller) SubscribeToEvents(handler func(ctx context.Context, event any)) func() {
	return c.eventBus.SubscribeToEvents(handler)
}

func NewDesiredStateChangedMessage(desiredState coretypes.MachineDesiredState) *DesiredStateChangedMessage {
	return &DesiredStateChangedMessage{
		DesiredState: desiredState,
		Timestamp:    time.Now(),
	}
}

func NewStateChangedMessage(state coretypes.MachineState) *StateChangedMessage {
	return &StateChangedMessage{
		State:     state,
		Timestamp: time.Now(),
	}
}
