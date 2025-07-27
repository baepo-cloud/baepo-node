package runtime

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/vmruntime/internal/chclient"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func (r *Runtime) startHypervisor(ctx context.Context) error {
	r.vmmCmd = exec.Command(r.config.CloudHypervisorBinary, "--api-socket", r.getHypervisorSocketPath())
	if err := r.vmmCmd.Start(); err != nil {
		return fmt.Errorf("failed to start cloud hypervisor: %w", err)
	}

	for retry := 0; true; retry++ {
		res, err := r.vmmClient.GetVmmPingWithResponse(ctx)
		if err == nil && res.StatusCode() == http.StatusOK {
			break
		} else if retry >= 10 {
			_ = syscall.Kill(r.vmmCmd.Process.Pid, syscall.SIGKILL)
			_ = os.Remove(r.getHypervisorSocketPath())
			return err
		}
		time.Sleep(100 * time.Microsecond)
	}
	return nil
}

func (r *Runtime) createVM(ctx context.Context) error {
	disksConfig := make([]chclient.DiskConfig, len(r.config.Containers))
	for index, volume := range r.config.Containers {
		disksConfig[index] = chclient.DiskConfig{
			Path:      volume.VolumePath,
			Readonly:  typeutil.Ptr(false),
			Direct:    typeutil.Ptr(true),
			NumQueues: typeutil.Ptr(1),
			QueueSize: typeutil.Ptr(128),
		}
	}

	_, err := r.vmmClient.CreateVM(ctx, chclient.VmConfig{
		Cpus: &chclient.CpusConfig{
			BootVcpus: int(r.config.Cpus),
			MaxVcpus:  int(r.config.Cpus),
		},
		Memory: &chclient.MemoryConfig{
			Size: int64(r.config.MemoryMB * 1024 * 1024), // convert Mib to bytes
		},
		Disks: &disksConfig,
		Net: &[]chclient.NetConfig{
			{
				Tap:       &r.config.Network.InterfaceName,
				Mac:       typeutil.Ptr(r.config.Network.MacAddress.String()),
				NumQueues: typeutil.Ptr(2),
				QueueSize: typeutil.Ptr(256),
			},
		},
		Vsock: &chclient.VsockConfig{
			Cid:    3,
			Socket: r.getInitDaemonSocketPath(),
		},
		Console: &chclient.ConsoleConfig{
			Mode: chclient.ConsoleConfigModeFile,
			File: typeutil.Ptr(r.getHypervisorLogPath()),
		},
		Payload: chclient.PayloadConfig{
			Kernel:    typeutil.Ptr(r.config.VMLinux),
			Initramfs: typeutil.Ptr(r.getInitRamFSPath()),
			Cmdline:   typeutil.Ptr("console=ttyS0 console=hvc0 root=/dev/vda rw rdinit=/init"),
		},
		Rng: &chclient.RngConfig{
			Src: "/dev/urandom",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create vm: %v", err)
	}

	return nil
}

func (r *Runtime) bootVM(ctx context.Context) error {
	res, err := r.vmmClient.BootVMWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("failed to boot vm: %w", err)
	} else if statusCode := res.StatusCode(); statusCode != http.StatusNoContent {
		return fmt.Errorf("failed to boot vm (status code %v): %v", statusCode, string(res.Body))
	}

	return nil
}

func (r *Runtime) terminateVM(ctx context.Context) error {
	_, err := r.vmmClient.DeleteVM(ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "connect: no such file or directory") {
			return fmt.Errorf("failed to delete vm: %w", err)
		}
	}

	_, err = r.vmmClient.ShutdownVMMWithResponse(ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "connect: no such file or directory") {
			return fmt.Errorf("failed to shutdown vmm: %w", err)
		}
	}

	if r.vmmCmd != nil && r.vmmCmd.Process != nil {
		_ = syscall.Kill(r.vmmCmd.Process.Pid, syscall.SIGKILL)
	}
	_ = os.Remove(r.getHypervisorSocketPath())
	_ = os.Remove(r.getInitRamFSPath())
	return nil
}
