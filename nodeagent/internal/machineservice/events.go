package machineservice

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"google.golang.org/protobuf/proto"
	"log/slog"
)

func (s *Service) newMachineEventsHandler(machine *types.Machine) func(context.Context, any) {
	return func(ctx context.Context, anyEvent any) {
		var machineEvent *types.MachineEvent
		switch event := anyEvent.(type) {
		case *corev1pb.MachineEvent:
			switch event.Event.(type) {
			case *corev1pb.MachineEvent_DesiredStateChanged:
				machineEvent = &types.MachineEvent{
					ID:        event.EventId,
					Type:      types.MachineEventTypeDesiredStateChanged,
					MachineID: machine.ID,
					Machine:   machine,
					Timestamp: event.Timestamp.AsTime(),
				}
			case *corev1pb.MachineEvent_StateChanged:
				machineEvent = &types.MachineEvent{
					ID:        event.EventId,
					Type:      types.MachineEventTypeStateChanged,
					MachineID: machine.ID,
					Machine:   machine,
					Timestamp: event.Timestamp.AsTime(),
				}
			}
		case *corev1pb.ContainerEvent:
			switch event.Event.(type) {
			case *corev1pb.ContainerEvent_StateChanged:
				var container *types.Container
				for _, current := range machine.Containers {
					if current.ID == event.ContainerId {
						container = current
						break
					}
				}
				if container == nil {
					return
				}

				machineEvent = &types.MachineEvent{
					ID:          event.EventId,
					Type:        types.MachineEventTypeContainerStateChanged,
					MachineID:   machine.ID,
					Machine:     machine,
					ContainerID: &container.ID,
					Container:   container,
					Timestamp:   event.Timestamp.AsTime(),
				}
			}
		}
		if machineEvent == nil {
			return
		}

		payloadBytes, err := proto.Marshal(anyEvent.(proto.Message))
		if err != nil {
			s.log.Error("failed to marshal machine event", slog.Any("error", err))
			return
		}

		machineEvent.Payload = payloadBytes
		if err = s.db.WithContext(ctx).Save(&machineEvent).Error; err != nil {
			s.log.Error("failed to insert machine event", slog.Any("error", err))
			return
		}

		fmt.Println(machineEvent)
	}
}
