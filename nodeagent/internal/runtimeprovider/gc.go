package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"os"
	"path"
	"strings"
)

func (p *Provider) GC(ctx context.Context, getExpectedMachineIDs func() []string) error {
	p.gcMutex.Lock()
	defer p.gcMutex.Unlock()

	runtimesDir := path.Join(p.config.StorageDirectory, "runtimes")
	entries, err := os.ReadDir(runtimesDir)
	if err != nil {
		return fmt.Errorf("failed to read runtimes directory: %w", err)
	}

	var machineIDs []string
	for _, entry := range entries {
		machineID, ok := strings.CutSuffix(entry.Name(), "_vm.socket")
		if entry.IsDir() || !ok {
			continue
		}

		machineIDs = append(machineIDs, machineID)
	}

	expectedMachineIDs := getExpectedMachineIDs()
	for _, machineID := range machineIDs {
		if typeutil.Includes(expectedMachineIDs, machineID) {
			continue
		}

		vmmClient, err := p.newCloudHypervisorHTTPClient(machineID)
		if err != nil {
			return fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
		}

		_, err = vmmClient.ShutdownVMMWithResponse(ctx)
		if err != nil {
			return fmt.Errorf("failed to shutdown vmm: %w", err)
		}

		_ = os.Remove(p.getInitRamFSPath(machineID))
		_ = os.Remove(p.getInitDaemonSocketPath(machineID))
	}

	return nil
}
