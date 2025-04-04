package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/pkg/chclient"
	"github.com/baepo-app/baepo-node/pkg/types"
	"github.com/baepo-app/baepo-node/pkg/typeutil"
)

func (p *Provider) Create(ctx context.Context, machine *types.Machine) (int, error) {
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
			BootVcpus: machine.Spec.Vcpus,
			MaxVcpus:  machine.Spec.Vcpus,
		},
		Memory: &chclient.MemoryConfig{
			Size: machine.Spec.Memory,
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
			Initramfs: typeutil.Ptr(p.initRamFSPath),
			Cmdline: typeutil.Ptr(fmt.Sprintf(
				"console=ttyS0 console=hvc0 root=/dev/vda rw rdinit=/sbin/init -- %v/24 %v", // todo: pass mask
				machine.NetworkInterface.IPAddress.String(),
				machine.NetworkInterface.MacAddress.String(),
			)),
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
