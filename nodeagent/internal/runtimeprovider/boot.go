package runtimeprovider

import (
	"context"
	"fmt"
)

func (p *Provider) Boot(ctx context.Context, machineID string) error {
	vmmClient, err := p.newCloudHypervisorHTTPClient(machineID)
	if err != nil {
		return fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	_, err = vmmClient.BootVM(ctx)
	if err != nil {
		return fmt.Errorf("failed to boot vm: %w", err)
	}

	return nil
}
