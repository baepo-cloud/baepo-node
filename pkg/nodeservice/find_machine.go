package nodeservice

import (
	"context"
	"github.com/baepo-app/baepo-node/pkg/types"
)

func (s *Service) FindMachine(ctx context.Context, machineID string) (*types.NodeMachine, error) {
	machine, ok := s.machines[machineID]
	if !ok {
		return nil, types.ErrNodeMachineNotFound
	}

	return machine, nil
}
