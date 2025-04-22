package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/runtimeprovider/chclient"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func (p *Provider) RecoverRunningMachines(ctx context.Context) ([]*types.Machine, error) {
	entries, err := os.ReadDir(filepath.Join(p.storageDirectory, "runtimes"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to read runtime directory: %w", err)
	}

	var recoveredMachines []*types.Machine
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "vm_") || !strings.HasSuffix(entry.Name(), ".socket") {
			continue
		}

		machineID := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "vm_"), ".socket")
		machine, err := p.recoverMachine(ctx, machineID)
		if err != nil {
			slog.Error("failed to recover machine",
				slog.String("machine-id", machineID),
				slog.Any("error", err))
			continue
		}

		recoveredMachines = append(recoveredMachines, machine)
	}

	return recoveredMachines, nil
}

func (p *Provider) recoverMachine(ctx context.Context, machineID string) (*types.Machine, error) {
	vmmClient, err := p.newCloudHypervisorHTTPClient(machineID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	vmInfo, err := vmmClient.GetVmInfoWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info: %w", err)
	}

	if vmInfo.JSON200 == nil || vmInfo.JSON200.State != chclient.Running {
		return nil, fmt.Errorf("VM is not in running state: %v", vmInfo.Status())
	}

	pingRes, err := vmmClient.GetVmmPingWithResponse(ctx)
	if err != nil || pingRes.JSON200 == nil || pingRes.JSON200.Pid == nil {
		return nil, fmt.Errorf("failed to get VM process ID: %w", err)
	}

	machine := &types.Machine{
		ID:         machineID,
		State:      types.MachineStateRunning,
		RuntimePID: typeutil.Ptr(int(*pingRes.JSON200.Pid)),
	}

	// Extract network interface details
	if vmInfo.JSON200.Config.Net != nil && len(*vmInfo.JSON200.Config.Net) > 0 {
		netConfig := (*vmInfo.JSON200.Config.Net)[0]

		if netConfig.Tap != nil {
			machine.NetworkInterface = &types.NetworkInterface{
				Name: *netConfig.Tap,
			}
			if netConfig.Mac != nil {
				machine.NetworkInterface.MacAddress, _ = net.ParseMAC(*netConfig.Mac)
			}
		}
	}

	if vmInfo.JSON200.Config.Disks != nil && len(*vmInfo.JSON200.Config.Disks) > 0 {
		diskConfig := (*vmInfo.JSON200.Config.Disks)[0]
		if diskConfig.Path != "" {
			machine.Volume = &types.Volume{
				ID:   filepath.Base(diskConfig.Path),
				Path: diskConfig.Path,
			}
			if diskConfig.Readonly != nil {
				machine.Volume.ReadOnly = *diskConfig.Readonly
			}
		}
	}

	if vmInfo.JSON200.Config.Cpus != nil {
		machine.Spec = &types.MachineSpec{
			Vcpus: uint32(vmInfo.JSON200.Config.Cpus.BootVcpus),
		}
		if vmInfo.JSON200.Config.Memory != nil {
			machine.Spec.MemoryMB = uint64(vmInfo.JSON200.Config.Memory.Size / 1024 / 1024)
		}
	}

	return machine, nil
}
