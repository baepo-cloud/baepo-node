package nodeservice

import (
	"context"
	"github.com/baepo-app/baepo-node/pkg/types"
)

func (s *Service) FindMachine(ctx context.Context, machineID string) (*types.Machine, error) {
	machine, ok := s.machines[machineID]
	if !ok {
		return nil, types.ErrMachineNotFound
	}

	return machine, nil
}
