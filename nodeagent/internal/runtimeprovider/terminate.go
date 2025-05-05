package runtimeprovider

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func (p *Provider) Terminate(ctx context.Context, machineID string) error {
	p.gcMutex.RLock()
	defer p.gcMutex.RUnlock()

	vmmClient, err := p.newCloudHypervisorHTTPClient(machineID)
	if err != nil {
		return fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	_, err = vmmClient.DeleteVM(ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "connect: no such file or directory") {
			return fmt.Errorf("failed to delete vm: %w", err)
		}
	}

	_, err = vmmClient.ShutdownVMMWithResponse(ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "connect: no such file or directory") {
			return fmt.Errorf("failed to shutdown vmm: %w", err)
		}
	}

	_ = os.Remove(p.getInitRamFSPath(machineID))
	_ = os.Remove(p.getInitDaemonSocketPath(machineID))
	return nil
}
