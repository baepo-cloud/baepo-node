package machineservice

import (
	"context"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
)

func (s *Service) UpdateDesiredState(ctx context.Context, opts types.MachineUpdateDesiredStateOptions) (*types.Machine, error) {
	ctrl, ok := s.machineControllers.Get(opts.MachineID)
	if !ok {
		return nil, types.ErrMachineNotFound
	}

	ctrl.SetDesiredState(opts.DesiredState)
	return ctrl.GetState().Machine, nil
}
