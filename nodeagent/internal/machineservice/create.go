package machineservice

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"log/slog"
)

func (s *Service) Create(ctx context.Context, opts types.MachineCreateOptions) (*types.Machine, error) {
	s.log.Info("requesting machine creation", slog.String("machine-id", opts.MachineID))
	machine := &types.Machine{
		ID:           opts.MachineID,
		State:        types.MachineStatePending,
		DesiredState: opts.DesiredState,
		Spec:         &opts.Spec,
		Containers:   make([]*types.Container, len(opts.Containers)),
	}
	for index, container := range opts.Containers {
		machine.Containers[index] = &types.Container{
			ID:        container.ContainerID,
			MachineID: machine.ID,
			Spec:      &container.Spec,
		}
	}
	if err := s.db.WithContext(ctx).Save(&machine).Error; err != nil {
		return nil, fmt.Errorf("failed to create machine: %w", err)
	}

	s.machineControllerLock.Lock()
	defer s.machineControllerLock.Unlock()
	s.machineControllers[machine.ID] = s.newMachineController(machine)
	return machine, nil
}
