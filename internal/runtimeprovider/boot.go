package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
)

func (p *Provider) Boot(ctx context.Context, machine *types.Machine) error {
	vmmClient, err := p.newCloudHypervisorHTTPClient(machine.ID)
	if err != nil {
		return fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	_, err = vmmClient.BootVM(ctx)
	if err != nil {
		return fmt.Errorf("failed to boot vm: %w", err)
	}

	return nil
}
