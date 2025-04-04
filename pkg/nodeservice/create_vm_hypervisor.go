package nodeservice

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func (s *Service) CreateVMHypervisor(ctx context.Context, machineID string) (int, error) {
	socketPath := s.getHypervisorSocketPath(machineID)
	cmd := exec.Command(s.binaryPath, "--api-socket", socketPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("failed to start cloud hypervisor: %w", err)
	}

	vmmClient, err := s.newCloudHypervisorHTTPClient(machineID)
	if err != nil {
		return -1, fmt.Errorf("failed to create cloud hypervisor http client: %w", err)
	}

	for retry := 0; true; retry++ {
		res, err := vmmClient.GetVmmPingWithResponse(ctx)
		if err == nil && res.StatusCode() == http.StatusOK {
			break
		} else if retry >= 10 {
			_ = syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)
			_ = os.Remove(socketPath)
			return -1, err
		}
		time.Sleep(100 * time.Microsecond)
	}
	return cmd.Process.Pid, nil
}
