package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/nodeservice/machinecontroller"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"log/slog"
)

func (s *Service) loadMachines(ctx context.Context) error {
	slog.Info("registering node...")

	var machines []*types.Machine
	err := s.db.WithContext(ctx).
		Joins("Volume").
		Joins("NetworkInterface").
		Where("machines.state NOT IN ?", []types.MachineState{types.MachineStateTerminated}).
		Find(&machines).
		Error
	if err != nil {
		return fmt.Errorf("failed to retrieve machines: %w", err)
	}

	s.machineControllerLock.Lock()
	defer s.machineControllerLock.Unlock()

	for _, machine := range machines {
		s.machineControllers[machine.ID] = s.newMachineController(machine)
	}

	return nil
}

func (s *Service) newMachineController(machine *types.Machine) *machinecontroller.Controller {
	ctrl := machinecontroller.New(s.db, s.volumeProvider, s.networkProvider, s.runtimeProvider, machine)

	return ctrl
}

func (s *Service) FindMachine(ctx context.Context, machineID string) (*types.Machine, error) {
	s.machineControllerLock.RLock()
	defer s.machineControllerLock.RUnlock()

	ctrl, ok := s.machineControllers[machineID]
	if !ok {
		return nil, types.ErrMachineNotFound
	}

	return ctrl.GetMachine(), nil
}

func (s *Service) StartMachine(ctx context.Context, opts types.NodeStartMachineOptions) (*types.Machine, error) {
	s.log.Info("requesting machine start", slog.String("machine-id", opts.MachineID))

	machine := &types.Machine{
		ID:           opts.MachineID,
		State:        types.MachineStatePending,
		DesiredState: types.MachineDesiredStateRunning,
		Spec: &types.MachineSpec{
			Image:    opts.Spec.Image,
			Vcpus:    opts.Spec.Vcpus,
			MemoryMB: opts.Spec.MemoryMB,
			Env:      map[string]string{},
		},
	}
	if err := s.db.WithContext(ctx).Create(&machine).Error; err != nil {
		return nil, fmt.Errorf("failed to create machine: %w", err)
	}

	s.machineControllerLock.Lock()
	defer s.machineControllerLock.Unlock()
	s.machineControllers[machine.ID] = s.newMachineController(machine)

	return machine, nil
}

func (s *Service) StopMachine(ctx context.Context, machineID string) (*types.Machine, error) {
	s.machineControllerLock.RLock()
	defer s.machineControllerLock.RUnlock()

	ctrl, ok := s.machineControllers[machineID]
	if !ok {
		return nil, types.ErrMachineNotFound
	}

	s.log.Info("requesting machine stop", slog.String("machine-id", machineID))
	ctrl.UpdateDesiredState(types.MachineDesiredStateTerminated)
	return ctrl.GetMachine(), nil
}
