package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/pkg/types"
)

func (s *Service) BootVM(ctx context.Context, machine *types.NodeMachine) error {
	vmmClient, err := s.newCloudHypervisorHTTPClient(machine.MachineID)
	if err != nil {
		return fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	_, err = vmmClient.BootVM(ctx)
	if err != nil {
		return fmt.Errorf("failed to boot vm: %w", err)
	}

	return nil
}
