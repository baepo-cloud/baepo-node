package registrationservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/v1pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"github.com/nrednav/cuid2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (c *Connection) startMachineEventListener(ctx context.Context) error {
	events := c.service.machineService.SubscribeToEvents(ctx)
	go func() {
		for event := range events {
			if err := c.sendMachineEvent(ctx, event); err != nil {
				c.log.Error("failed to send machine event", slog.Any("error", err))
			}
		}
	}()

	return nil
}

func (c *Connection) syncMachines(ctx context.Context, machines []*apiv1pb.NodeControllerServerEvent_Machine) error {
	c.log.Info("syncing machines", slog.Int("count", len(machines)))

	// syncing expected machines
	expectedMachines := map[string]bool{}
	for _, spec := range machines {
		machine, err := c.syncMachineFromSpec(ctx, spec)
		if err != nil {
			return fmt.Errorf("failed to reconcile machine: %w", err)
		}

		expectedMachines[spec.MachineId] = machine != nil
	}

	// now, terminating machines that are not expected to be running
	currentMachines, err := c.service.machineService.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list machines: %w", err)
	}

	for _, machine := range currentMachines {
		if ok := expectedMachines[machine.ID]; ok {
			continue
		}

		_, err = c.service.machineService.UpdateDesiredState(ctx, types.MachineUpdateDesiredStateOptions{
			MachineID:    machine.ID,
			DesiredState: coretypes.MachineDesiredStateTerminated,
		})
		if err != nil && !errors.Is(err, types.ErrMachineNotFound) {
			return fmt.Errorf("failed to terminate machine: %w", err)
		}
	}
	return nil
}

func (c *Connection) syncMachineFromSpec(ctx context.Context, spec *apiv1pb.NodeControllerServerEvent_Machine) (*types.Machine, error) {
	desiredState := v1pbadapter.ToMachineDesiredState(spec.DesiredState)
	log := c.log.With(slog.String("machine-id", spec.MachineId), slog.Any("desired-state", desiredState))
	machine, err := c.service.machineService.FindByID(ctx, spec.MachineId)
	shouldSendFakeTerminationEvents := false
	if errors.Is(err, types.ErrMachineNotFound) {
		if spec.DesiredState == corev1pb.MachineDesiredState_MachineDesiredState_Terminated {
			shouldSendFakeTerminationEvents = true
		} else {
			log.Info("missing machine, creating")
			machine, err = c.createMachine(ctx, spec)
			if err != nil {
				return nil, fmt.Errorf("failed to create machine: %w", err)
			}
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to find machine: %w", err)
	} else if current := machine.DesiredState; current != desiredState {
		log.Info("desired state mismatch, updating", slog.Any("current-desired-state", current))
		machine, err = c.service.machineService.UpdateDesiredState(ctx, types.MachineUpdateDesiredStateOptions{
			MachineID:    spec.MachineId,
			DesiredState: desiredState,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update machine desired state: %w", err)
		}
	}

	previousEvents, err := c.service.machineService.ListEvents(ctx, spec.MachineId)
	if err != nil {
		return nil, fmt.Errorf("failed to list previous machine events: %w", err)
	}

	for _, event := range previousEvents {
		if err = c.sendMachineEvent(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to send machine event: %w", err)
		}
	}
	if shouldSendFakeTerminationEvents {
		err = c.stream.Send(&apiv1pb.NodeControllerClientEvent{
			Event: &apiv1pb.NodeControllerClientEvent_Machine{
				Machine: &corev1pb.MachineEvent{
					EventId: cuid2.Generate(),
					Event: &corev1pb.MachineEvent_DesiredStateChanged{
						DesiredStateChanged: &corev1pb.MachineEvent_DesiredStateChangedEvent{
							DesiredState: corev1pb.MachineDesiredState_MachineDesiredState_Terminated,
						},
					},
					MachineId: spec.MachineId,
					Timestamp: timestamppb.Now(),
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to send desired state machine event: %w", err)
		}

		err = c.stream.Send(&apiv1pb.NodeControllerClientEvent{
			Event: &apiv1pb.NodeControllerClientEvent_Machine{
				Machine: &corev1pb.MachineEvent{
					EventId: cuid2.Generate(),
					Event: &corev1pb.MachineEvent_StateChanged{
						StateChanged: &corev1pb.MachineEvent_StateChangedEvent{
							State: corev1pb.MachineState_MachineState_Terminated,
						},
					},
					MachineId: spec.MachineId,
					Timestamp: timestamppb.Now(),
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to send state changed machine event: %w", err)
		}
	}

	return machine, nil
}

func (c *Connection) createMachine(ctx context.Context, machine *apiv1pb.NodeControllerServerEvent_Machine) (*types.Machine, error) {
	opts := types.MachineCreateOptions{
		MachineID:    machine.MachineId,
		DesiredState: v1pbadapter.ToMachineDesiredState(machine.DesiredState),
		Spec:         (*types.MachineSpec)(v1pbadapter.ToMachineSpec(machine.Spec)),
		Containers:   make([]types.MachineCreateContainerOptions, len(machine.Containers)),
	}
	for index, container := range machine.Containers {
		opts.Containers[index] = types.MachineCreateContainerOptions{
			ContainerID: container.ContainerId,
			Spec:        v1pbadapter.ToContainerSpec(container.Spec),
		}
	}

	return c.service.machineService.Create(ctx, opts)
}

func (c *Connection) sendMachineEvent(ctx context.Context, event *types.MachineEvent) error {
	anyProto, err := event.ProtoPayload()
	if err != nil {
		return err
	}

	switch proto := anyProto.(type) {
	case *corev1pb.MachineEvent:
		return c.stream.Send(&apiv1pb.NodeControllerClientEvent{
			Event: &apiv1pb.NodeControllerClientEvent_Machine{
				Machine: proto,
			},
		})
	case *corev1pb.ContainerEvent:
		return c.stream.Send(&apiv1pb.NodeControllerClientEvent{
			Event: &apiv1pb.NodeControllerClientEvent_Container{
				Container: proto,
			},
		})
	default:
		return nil
	}
}
