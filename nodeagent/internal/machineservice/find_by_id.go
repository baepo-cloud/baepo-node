package machineservice

import (
	"context"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
)

func (s *Service) FindByID(ctx context.Context, machineID string) (*types.Machine, error) {
	s.machineControllerLock.RLock()
	defer s.machineControllerLock.RUnlock()

	ctrl, ok := s.machineControllers[machineID]
	if !ok {
		return nil, types.ErrMachineNotFound
	}

	return ctrl.GetMachine(), nil
}
