package runtimeprovider

import (
	"context"
	"fmt"
	"net/http"
)

func (p *Provider) Boot(ctx context.Context, machineID string) error {
	vmmClient, err := p.newCloudHypervisorHTTPClient(machineID)
	if err != nil {
		return fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	res, err := vmmClient.BootVMWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("failed to boot vm: %w", err)
	} else if statusCode := res.StatusCode(); statusCode != http.StatusNoContent {
		return fmt.Errorf("failed to boot vm (status code %v): %v", statusCode, string(res.Body))
	}

	return nil
}
