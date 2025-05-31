package machineservice

import (
	"context"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
)

func (s *Service) List(ctx context.Context) ([]*types.Machine, error) {
	s.machineControllerLock.RLock()
	defer s.machineControllerLock.RUnlock()

	var machines []*types.Machine
	for _, ctrl := range s.machineControllers {
		machines = append(machines, ctrl.GetMachine())
	}
	return machines, nil
}
