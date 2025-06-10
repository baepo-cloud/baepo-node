package registrationservice

import (
	"context"
	"errors"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/v1pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	"log/slog"
)

func (s *Service) syncMachines(ctx context.Context, machines []*apiv1pb.NodeControllerServerEvent_Machine) error {
	s.log.Info("processing expected machines list")
	expectedMachines := map[string]bool{}
	for _, spec := range machines {
		expectedMachines[spec.MachineId] = true
		if err := s.reconcileWithExpectedMachine(ctx, spec); err != nil {
			return fmt.Errorf("failed to reconcile machine: %w", err)
		}
	}

	currentMachines, err := s.machineService.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list machines: %w", err)
	}

	for _, machine := range currentMachines {
		if _, ok := expectedMachines[machine.ID]; ok {
			continue
		}

		_, err = s.machineService.UpdateDesiredState(ctx, types.MachineUpdateDesiredStateOptions{
			MachineID:    machine.ID,
			DesiredState: coretypes.MachineDesiredStateTerminated,
		})
		if err != nil && !errors.Is(err, types.ErrMachineNotFound) {
			return fmt.Errorf("failed to terminate machine: %w", err)
		}
	}
	return nil
}

func (s *Service) reconcileWithExpectedMachine(ctx context.Context, spec *apiv1pb.NodeControllerServerEvent_Machine) error {
	desiredState := v1pbadapter.ToMachineDesiredState(spec.DesiredState)
	log := s.log.With(slog.String("machine-id", spec.MachineId), slog.Any("desired-state", desiredState))
	machine, err := s.machineService.FindByID(ctx, spec.MachineId)
	if errors.Is(err, types.ErrMachineNotFound) {
		log.Info("missing machine, creating")
		if err = s.createMachine(ctx, spec); err != nil {
			return fmt.Errorf("failed to create machine: %w", err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("failed to find machine: %w", err)
	} else if current := machine.DesiredState; current != desiredState {
		log.Info("desired state mismatch, updating", slog.Any("current-desired-state", current))
		_, err = s.machineService.UpdateDesiredState(ctx, types.MachineUpdateDesiredStateOptions{
			MachineID:    spec.MachineId,
			DesiredState: desiredState,
		})
		if err != nil {
			return fmt.Errorf("failed to update machine desired state: %w", err)
		}

		return nil
	}

	return nil
}

func (s *Service) createMachine(ctx context.Context, machine *apiv1pb.NodeControllerServerEvent_Machine) error {
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

	_, err := s.machineService.Create(ctx, opts)
	return err
}
