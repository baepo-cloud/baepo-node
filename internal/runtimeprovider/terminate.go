package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"log/slog"
	"os"
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
	_ = os.Remove(p.getInitRamFSPath(machine.ID))
	return err
}
