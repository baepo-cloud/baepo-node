package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/runtimeprovider/chclient"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
)

func (p *Provider) Create(ctx context.Context, machine *types.Machine) (int, error) {
	err := p.BuildInitRamFS(ctx, machine)
	if err != nil {
		return 0, err
	}

	pid, err := p.StartHypervisor(ctx, machine.ID)
	if err != nil {
		return -1, err
	}

	vmmClient, err := p.newCloudHypervisorHTTPClient(machine.ID)
	if err != nil {
		return -1, fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	_, err = vmmClient.CreateVM(ctx, chclient.VmConfig{
		Cpus: &chclient.CpusConfig{
			BootVcpus: int(machine.Spec.Vcpus),
			MaxVcpus:  int(machine.Spec.Vcpus),
		},
		Memory: &chclient.MemoryConfig{
			Size: int64(machine.Spec.MemoryMB * 1024 * 1024), // convert Mib to bytes
		},
		Disks: &[]chclient.DiskConfig{
			{
				Path:      machine.Volume.Path,
				Readonly:  typeutil.Ptr(machine.Volume.ReadOnly),
				Direct:    typeutil.Ptr(true),
				NumQueues: typeutil.Ptr(1),
				QueueSize: typeutil.Ptr(128),
			},
		},
		Net: &[]chclient.NetConfig{
			{
				Tap:       &machine.NetworkInterface.Name,
				Mac:       typeutil.Ptr(machine.NetworkInterface.MacAddress.String()),
				NumQueues: typeutil.Ptr(2),
				QueueSize: typeutil.Ptr(256),
			},
		},
		Vsock: &chclient.VsockConfig{
			Cid:    3,
			Socket: p.getInitDaemonSocketPath(machine.ID),
		},
		Console: &chclient.ConsoleConfig{
			Mode: chclient.ConsoleConfigModeFile,
			File: typeutil.Ptr(p.getHypervisorLogPath(machine.ID)),
		},
		Payload: chclient.PayloadConfig{
			Kernel:    typeutil.Ptr(p.vmLinuxPath),
			Initramfs: typeutil.Ptr(p.getInitRamFSPath(machine.ID)),
			Cmdline:   typeutil.Ptr("console=ttyS0 console=hvc0 root=/dev/vda rw rdinit=/init"),
		},
		Rng: &chclient.RngConfig{
			Src: "/dev/urandom",
		},
	})
	if err != nil {
		return -1, fmt.Errorf("failed to create vm: %v", err)
	}

	return pid, nil
}
