package runtimeprovider

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

func (p *Provider) Terminate(ctx context.Context, machineID string) error {
	p.gcMutex.RLock()
	defer p.gcMutex.RUnlock()

	vmmClient, err := p.newCloudHypervisorHTTPClient(machineID)
	if err != nil {
		return fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	if _, err = vmmClient.DeleteVM(ctx); err != nil {
		slog.Warn("failed to delete vm", slog.Any("error", err))
	}

	_, err = vmmClient.ShutdownVMMWithResponse(ctx)
	_ = os.Remove(p.getInitRamFSPath(machineID))
	_ = os.Remove(p.getInitDaemonSocketPath(machineID))
	return err
}
