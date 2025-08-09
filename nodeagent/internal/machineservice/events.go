package machineservice

import (
	"context"
	"fmt"
	"log/slog"

	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/v1pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/machineservice/machinecontroller"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"github.com/nrednav/cuid2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm/clause"
)

func (s *Service) handleMachineEventsStorage(machine *types.Machine) func(context.Context, any) {
	return func(ctx context.Context, anyEvent any) {
		var machineEvent *types.MachineEvent
		var protoMessage proto.Message
		var container *types.Container

		switch event := anyEvent.(type) {
		case *machinecontroller.DesiredStateChangedMessage:
			machineEvent = &types.MachineEvent{
				ID:        cuid2.Generate(),
				Type:      types.MachineEventTypeDesiredStateChanged,
				MachineID: machine.ID,
				Timestamp: event.Timestamp,
			}
			protoMessage = &corev1pb.MachineEvent{
				EventId:   machineEvent.ID,
				MachineId: machine.ID,
				Timestamp: timestamppb.New(event.Timestamp),
				Event: &corev1pb.MachineEvent_DesiredStateChanged{
					DesiredStateChanged: &corev1pb.MachineEvent_DesiredStateChangedEvent{
						DesiredState: v1pbadapter.FromMachineDesiredState(event.DesiredState),
					},
				},
			}
		case *machinecontroller.StateChangedMessage:
			machineEvent = &types.MachineEvent{
				ID:        cuid2.Generate(),
				Type:      types.MachineEventTypeStateChanged,
				MachineID: machine.ID,
				Timestamp: event.Timestamp,
			}
			protoMessage = &corev1pb.MachineEvent{
				EventId:   machineEvent.ID,
				MachineId: machine.ID,
				Timestamp: timestamppb.New(event.Timestamp),
				Event: &corev1pb.MachineEvent_StateChanged{
					StateChanged: &corev1pb.MachineEvent_StateChangedEvent{
						State: v1pbadapter.FromMachineState(event.State),
					},
				},
			}
		case *machinecontroller.ContainerStateChangedMessage:
			for _, current := range machine.Containers {
				if current.ID == event.Event.ContainerId {
					container = current
					break
				}
			}
			if container == nil {
				return
			}

			machineEvent = &types.MachineEvent{
				ID:          event.EventID,
				Type:        types.MachineEventTypeContainerStateChanged,
				MachineID:   machine.ID,
				ContainerID: &container.ID,
				Timestamp:   event.Timestamp,
			}
			protoMessage = &corev1pb.ContainerEvent{
				EventId:     event.EventID,
				ContainerId: container.ID,
				Timestamp:   timestamppb.New(event.Timestamp),
				Event: &corev1pb.ContainerEvent_StateChanged{
					StateChanged: &corev1pb.ContainerEvent_StateChangedEvent{
						State:            event.Event.State,
						StartedAt:        event.Event.StartedAt,
						ExitedAt:         event.Event.ExitedAt,
						ExitCode:         event.Event.ExitCode,
						ExitError:        event.Event.ExitError,
						Healthy:          event.Event.Healthy,
						HealthcheckError: event.Event.HealthcheckError,
						RestartCount:     event.Event.RestartCount,
					},
				},
			}
		}
		if machineEvent == nil {
			return
		}

		payloadBytes, err := proto.Marshal(protoMessage)
		if err != nil {
			s.log.Error("failed to marshal machine event payload", slog.Any("error", err))
			return
		}

		machineEvent.Payload = payloadBytes
		err = s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&machineEvent).Error
		if err != nil {
			s.log.Error("failed to insert machine event", slog.Any("error", err))
			return
		}

		machineEvent.Machine = machine
		machineEvent.Container = container
		s.machineEvents.PublishEvent(machineEvent)
	}
}

func (s *Service) handleMachineTerminated(machine *types.Machine) func(context.Context, any) {
	return func(ctx context.Context, anyEvent any) {
		event, ok := anyEvent.(*machinecontroller.StateChangedMessage)
		if !ok || event.State != coretypes.MachineStateTerminated {
			return
		}

		controller, ok := s.machineControllers.Get(machine.ID)
		if !ok {
			return
		}

		if err := controller.Stop(); err != nil {
			s.log.Error("failed to stop machine controller",
				slog.String("machine-id", machine.ID),
				slog.Any("error", err))
			return
		}

		s.machineControllers.Del(machine.ID)
	}
}

func (s *Service) ListEvents(ctx context.Context, machineID string) ([]*types.MachineEvent, error) {
	var events []*types.MachineEvent
	err := s.db.WithContext(ctx).Find(&events, "machine_id = ?", machineID).Order("timestamp").Error
	if err != nil {
		return nil, fmt.Errorf("could not list machine events: %w", err)
	}

	return events, nil
}

func (s *Service) SubscribeToEvents(ctx context.Context) <-chan *types.MachineEvent {
	eventsChan := make(chan *types.MachineEvent)
	closeSubscriber := s.machineEvents.SubscribeToEvents(func(_ context.Context, event *types.MachineEvent) {
		go func() {
			select {
			case <-ctx.Done():
				return
			default:
				eventsChan <- event
			}
		}()
	})

	go func() {
		<-ctx.Done()
		closeSubscriber()
		close(eventsChan)
	}()

	return eventsChan
}
