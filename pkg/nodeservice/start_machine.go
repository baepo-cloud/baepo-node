package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/pkg/types"
	"log/slog"
)

func (s *Service) StartMachine(ctx context.Context, opts types.NodeStartMachineOptions) (*types.Machine, error) {
	slog.Info("starting machine", slog.String("machine-id", opts.MachineID))
	machine := &types.Machine{
		ID: opts.MachineID,
		Spec: &types.MachineSpec{
			Vcpus:  opts.Spec.Vcpus,
			Memory: opts.Spec.Memory,
			Env:    opts.Spec.Env,
		},
	}

	volume, err := s.volumeProvider.CreateVolume(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine volume: %w", err)
	}
	machine.Volume = volume

	machineNetwork, err := s.networkProvider.AllocateInterface()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate machine network: %w", err)
	}
	machine.NetworkInterface = machineNetwork

	machine.RuntimePID, err = s.runtimeProvider.Create(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %w", err)
	}

	err = s.runtimeProvider.Boot(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("failed to boot machine: %w", err)
	}

	s.lock.Lock()
	s.machines[machine.ID] = machine
	s.lock.Unlock()

	return machine, nil
}
