package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/runtimeprovider/chclient"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
)

func (p *Provider) Create(ctx context.Context, opts types.RuntimeCreateOptions) (int, error) {
	err := p.BuildInitRamFS(ctx, opts)
	if err != nil {
		return 0, err
	}

	pid, err := p.StartHypervisor(ctx, opts.MachineID)
	if err != nil {
		return -1, err
	}

	vmmClient, err := p.newCloudHypervisorHTTPClient(opts.MachineID)
	if err != nil {
		return -1, fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	_, err = vmmClient.CreateVM(ctx, chclient.VmConfig{
		Cpus: &chclient.CpusConfig{
			BootVcpus: int(opts.Spec.Vcpus),
			MaxVcpus:  int(opts.Spec.Vcpus),
		},
		Memory: &chclient.MemoryConfig{
			Size: int64(opts.Spec.MemoryMB * 1024 * 1024), // convert Mib to bytes
		},
		Disks: &[]chclient.DiskConfig{
			{
				Path:      opts.Volume.Path,
				Readonly:  typeutil.Ptr(opts.Volume.ReadOnly),
				Direct:    typeutil.Ptr(true),
				NumQueues: typeutil.Ptr(1),
				QueueSize: typeutil.Ptr(128),
			},
		},
		Net: &[]chclient.NetConfig{
			{
				Tap:       &opts.NetworkInterface.Name,
				Mac:       typeutil.Ptr(opts.NetworkInterface.MacAddress.String()),
				NumQueues: typeutil.Ptr(2),
				QueueSize: typeutil.Ptr(256),
			},
		},
		Vsock: &chclient.VsockConfig{
			Cid:    3,
			Socket: p.getInitDaemonSocketPath(opts.MachineID),
		},
		Console: &chclient.ConsoleConfig{
			Mode: chclient.ConsoleConfigModeFile,
			File: typeutil.Ptr(p.getHypervisorLogPath(opts.MachineID)),
		},
		Payload: chclient.PayloadConfig{
			Kernel:    typeutil.Ptr(p.vmLinuxPath),
			Initramfs: typeutil.Ptr(p.getInitRamFSPath(opts.MachineID)),
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
