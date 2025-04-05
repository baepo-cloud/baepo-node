package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/internal/types"
	"log/slog"
)

func (p *Provider) Terminate(ctx context.Context, machine *types.Machine) error {
	vmmClient, err := p.newCloudHypervisorHTTPClient(machine.ID)
	if err != nil {
		return fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	if _, err = vmmClient.DeleteVM(ctx); err != nil {
		slog.Warn("failed to delete vm", slog.Any("error", err))
	}

	_, err = vmmClient.ShutdownVMMWithResponse(ctx)
	return err
}
