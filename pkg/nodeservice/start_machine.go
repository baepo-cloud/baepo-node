package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/pkg/types"
)

func (s *Service) StartMachine(ctx context.Context, opts types.NodeStartMachineOptions) (*types.NodeMachine, error) {
	machine := &types.NodeMachine{
		MachineID: opts.MachineID,
		Spec: &types.NodeMachineSpec{
			Vcpus:  opts.Spec.Vcpus,
			Memory: opts.Spec.Memory,
			Env:    opts.Spec.Env,
		},
	}

	volume, err := s.CreateVolume(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine volume: %w", err)
	}
	machine.Volume = volume

	hypervisorPID, err := s.CreateVMHypervisor(ctx, machine.MachineID)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine hypervisor: %w", err)
	}
	machine.HypervisorPID = hypervisorPID

	machineNetwork, err := s.AllocateMachineNetwork()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate machine network: %w", err)
	}
	machine.NetworkInterface = machineNetwork

	err = s.CreateVM(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %w", err)
	}

	err = s.BootVM(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("failed to boot machine: %w", err)
	}

	s.lock.Lock()
	s.machines[machine.MachineID] = machine
	s.lock.Unlock()

	return machine, nil
}
