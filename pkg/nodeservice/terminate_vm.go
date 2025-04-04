package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/pkg/types"
	"log/slog"
)

func (s *Service) TerminateVM(ctx context.Context, machine *types.NodeMachine) error {
	vmmClient, err := s.newCloudHypervisorHTTPClient(machine.MachineID)
	if err != nil {
		return fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	if _, err = vmmClient.DeleteVM(ctx); err != nil {
		slog.Warn("failed to delete vm", slog.Any("error", err))
	}

	_, err = vmmClient.ShutdownVMMWithResponse(ctx)
	return err
}
