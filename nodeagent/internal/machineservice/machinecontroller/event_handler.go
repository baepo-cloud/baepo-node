package machinecontroller

import (
	"context"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/v1pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"github.com/nrednav/cuid2"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"
)

func (c *Controller) handleEvent(ctx context.Context, payload any) {
	switch typedPayload := payload.(type) {
	case *corev1pb.MachineEvent:
		switch typedEvent := typedPayload.Event.(type) {
		case *corev1pb.MachineEvent_DesiredStateChanged:
			c.handleDesiredStateChange(ctx, typedEvent)
		case *corev1pb.MachineEvent_StateChanged:
			c.handleStateChange(ctx, typedEvent)
		}
	}
}

func (c *Controller) handleDesiredStateChange(ctx context.Context, event *corev1pb.MachineEvent_DesiredStateChanged) {
	desiredState := v1pbadapter.ToMachineDesiredState(event.DesiredStateChanged.DesiredState)
	err := c.updateMachine(func(machine *types.Machine) error {
		machine.DesiredState = desiredState
		return c.db.WithContext(ctx).Select("DesiredState").Save(machine).Error
	})
	if err != nil {
		c.log.Error("failed to update machine desired state",
			slog.String("desired-state", string(desiredState)),
			slog.Any("error", err))
	}

	machine := c.GetMachine()
	if !matchDesiredState(machine.State, machine.DesiredState) {
		go c.startReconciliation()
	}
}

func (c *Controller) handleStateChange(ctx context.Context, event *corev1pb.MachineEvent_StateChanged) {
	state := v1pbadapter.ToMachineState(event.StateChanged.State)
	err := c.updateMachine(func(machine *types.Machine) error {
		machine.State = state
		return c.db.WithContext(ctx).Select("State").Save(machine).Error
	})
	if err != nil {
		c.log.Error("failed to update machine state", slog.Any("state", state), slog.Any("error", err))
	}

	machine := c.GetMachine()
	if !matchDesiredState(machine.State, machine.DesiredState) {
		go c.startReconciliation()
	}

	c.syncInitEventsListener()
}

func (c *Controller) dispatchMachineStateChangeEvent(state coretypes.MachineState) {
	machine := c.GetMachine()
	if machine.State != state {
		c.eventBus.PublishEvent(&corev1pb.MachineEvent{
			EventId:   cuid2.Generate(),
			MachineId: machine.ID,
			Event: &corev1pb.MachineEvent_StateChanged{
				StateChanged: &corev1pb.MachineEvent_StateChangedEvent{
					State: v1pbadapter.FromMachineState(state),
				},
			},
			Timestamp: timestamppb.Now(),
		})
	}
}
