package machineservice

import (
	"context"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/machineservice/machinecontroller"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
)

func (s *Service) List(ctx context.Context) ([]*types.Machine, error) {
	var machines []*types.Machine
	s.machineControllers.ForEach(func(_ string, ctrl *machinecontroller.Controller) bool {
		machines = append(machines, ctrl.GetMachine())
		return true
	})
	return machines, nil
}
