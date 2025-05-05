package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/runtimeprovider/chclient"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
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

	disksConfig := make([]chclient.DiskConfig, len(opts.Volumes))
	for _, volume := range opts.Volumes {
		disksConfig[volume.Position] = chclient.DiskConfig{
			Path:      volume.Volume.Path,
			Readonly:  typeutil.Ptr(false),
			Direct:    typeutil.Ptr(true),
			NumQueues: typeutil.Ptr(1),
			QueueSize: typeutil.Ptr(128),
		}
	}

	_, err = vmmClient.CreateVM(ctx, chclient.VmConfig{
		Cpus: &chclient.CpusConfig{
			BootVcpus: int(opts.Spec.Cpus),
			MaxVcpus:  int(opts.Spec.Cpus),
		},
		Memory: &chclient.MemoryConfig{
			Size: int64(opts.Spec.MemoryMB * 1024 * 1024), // convert Mib to bytes
		},
		Disks: &disksConfig,
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
			Kernel:    typeutil.Ptr(p.config.VMLinux),
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
