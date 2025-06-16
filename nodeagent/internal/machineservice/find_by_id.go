package machineservice

import (
	"context"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
)

func (s *Service) FindByID(ctx context.Context, machineID string) (*types.Machine, error) {
	ctrl, ok := s.machineControllers.Get(machineID)
	if !ok {
		return nil, types.ErrMachineNotFound
	}

	return ctrl.GetState().Machine, nil
}
