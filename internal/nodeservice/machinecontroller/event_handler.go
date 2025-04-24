package machinecontroller

import (
	"context"
	"github.com/baepo-cloud/baepo-node/internal/pbadapter"
	"github.com/baepo-cloud/baepo-node/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"
)

func (c *Controller) handleEvent(ctx context.Context, unknownEvent *corev1pb.MachineEvent) {
	switch event := unknownEvent.Event.(type) {
	case *corev1pb.MachineEvent_DesiredStateChangedEvent:
		c.handleDesiredStateChange(ctx, event)
	case *corev1pb.MachineEvent_StateChangedEvent:
		c.handleStateChange(ctx, event)
	case *corev1pb.MachineEvent_HealthcheckEvent:
		c.handleHealthcheck(ctx, event)
	}
}

func (c *Controller) handleDesiredStateChange(ctx context.Context, event *corev1pb.MachineEvent_DesiredStateChangedEvent) {
	desiredState := pbadapter.ProtoToMachineDesiredState(event.DesiredStateChangedEvent.DesiredState)
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
	if !machine.State.MatchDesiredState(machine.DesiredState) {
		go c.startReconciliation()
	}
}

func (c *Controller) handleStateChange(ctx context.Context, event *corev1pb.MachineEvent_StateChangedEvent) {
	state := pbadapter.ProtoToMachineState(event.StateChangedEvent.State)
	err := c.updateMachine(func(machine *types.Machine) error {
		machine.State = state
		return c.db.WithContext(ctx).Select("State").Save(machine).Error
	})
	if err != nil {
		c.log.Error("failed to update machine state", slog.Any("state", state), slog.Any("error", err))
	}

	machine := c.GetMachine()
	if !machine.State.MatchDesiredState(machine.DesiredState) {
		go c.startReconciliation()
	}

	c.syncMonitoring()
}

func (c *Controller) handleHealthcheck(ctx context.Context, event *corev1pb.MachineEvent_HealthcheckEvent) {
	if event.HealthcheckEvent.Error != nil {
		c.log.Warn("machine healthcheck failed", slog.Any("error", *event.HealthcheckEvent.Error))
		c.monitoringConsecutiveErrorCount++

		if c.monitoringConsecutiveErrorCount >= 3 {
			c.dispatchMachineStateChangeEvent(types.MachineStateError)
		} else if c.monitoringConsecutiveErrorCount > 0 {
			c.dispatchMachineStateChangeEvent(types.MachineStateDegraded)
		}
		return
	}

	c.monitoringConsecutiveErrorCount = 0
	c.dispatchMachineStateChangeEvent(types.MachineStateRunning)
}

func (c *Controller) dispatchMachineStateChangeEvent(state types.MachineState) {
	machine := c.GetMachine()
	if machine.State != state {
		c.PublishEvent(&corev1pb.MachineEvent{
			Timestamp: timestamppb.Now(),
			MachineId: machine.ID,
			Event: &corev1pb.MachineEvent_StateChangedEvent{
				StateChangedEvent: &corev1pb.MachineEvent_StateChanged{
					State: pbadapter.MachineStateToProto(state),
				},
			},
		})
	}
}
