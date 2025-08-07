package runtime

import (
	"context"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/vsock"
	"github.com/baepo-cloud/baepo-node/vmruntime/internal/chclient"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"net"
	"net/http"
	"os/exec"
	"path"
	"time"
)

type (
	Config struct {
		coretypes.RuntimeConfig
		InitBinary            string
		InitContainerBinary   string
		CloudHypervisorBinary string
		VMLinux               string
		Debug                 bool
	}

	Runtime struct {
		config     *Config
		vmmClient  *chclient.ClientWithResponses
		vmmCmd     *exec.Cmd
		httpServer *http.Server
		logManager *logManager
	}
)

func New(config *Config) *Runtime {
	runtime := &Runtime{config: config}
	vmmClient, err := chclient.NewClientWithResponses(
		"http://localhost/api/v1",
		chclient.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", runtime.getHypervisorSocketPath())
				},
			},
		}),
	)
	if err != nil {
		panic(err)
	}

	runtime.vmmClient = vmmClient
	return runtime
}

func (r *Runtime) Start(ctx context.Context) error {
	if err := r.startConnectServer(); err != nil {
		return err
	}

	logManager, err := newLogManager(r)
	if err != nil {
		return fmt.Errorf("failed to initialize log manager: %w", err)
	}
	r.logManager = logManager

	if err = r.buildInitRamFS(ctx); err != nil {
		return err
	} else if err = r.startHypervisor(ctx); err != nil {
		return err
	} else if err = r.createVM(ctx); err != nil {
		return err
	} else if err = r.bootVM(ctx); err != nil {
		return err
	}

	return r.vmmCmd.Wait()
}

func (r *Runtime) Stop(ctx context.Context) error {
	if r.httpServer != nil {
		_ = r.httpServer.Shutdown(ctx)
	}

	if err := r.terminateVM(ctx); err != nil {
		return fmt.Errorf("failed to terminate vm: %w", err)
	}

	return r.ForceStop(ctx)
}

func (r *Runtime) ForceStop(ctx context.Context) error {
	if err := r.stopHypervisor(ctx); err != nil {
		return fmt.Errorf("failed to stop hypervisor: %w", err)
	}

	if r.logManager != nil {
		r.logManager.Stop()
	}

	return nil
}

func (r *Runtime) newInitClient() (nodev1pbconnect.InitClient, func()) {
	var conns []net.Conn
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				conn, err := vsock.DialContext(ctx, r.getInitDaemonSocketPath(), coretypes.InitServerPort)
				if err != nil {
					return nil, err
				}

				conns = append(conns, conn)
				return conn, nil
			},
			IdleConnTimeout:       10 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
		},
	}
	return nodev1pbconnect.NewInitClient(httpClient, "http://init"), func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}
}

func (r *Runtime) getHypervisorSocketPath() string {
	return path.Join(r.config.WorkingDir, "vmm.socket")
}

func (r *Runtime) getInitDaemonSocketPath() string {
	return path.Join(r.config.WorkingDir, "init.socket")
}

func (r *Runtime) getInitRamFSPath() string {
	return path.Join(r.config.WorkingDir, "initramfs.gz")
}

func (r *Runtime) getInitConfigPath() string {
	return path.Join(r.config.WorkingDir, "initconfig.json")
}
