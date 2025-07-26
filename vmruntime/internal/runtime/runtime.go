package runtime

import (
	"context"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/vsock"
	"github.com/baepo-cloud/baepo-node/vmruntime/internal/chclient"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"net"
	"net/http"
	"path"
	"time"
)

type (
	Config struct {
		coretypes.RuntimeConfig
		InitBinaryPath            string
		InitContainerBinaryPath   string
		CloudHypervisorBinaryPath string
		VMLinuxPath               string
	}

	Runtime struct {
		config    *Config
		vmmClient *chclient.ClientWithResponses
		vmmPID    *int
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

func (r *Runtime) getHypervisorLogPath() string {
	return path.Join(r.config.WorkingDir, "machine.log")
}

func (r *Runtime) getInitRamFSPath() string {
	return path.Join(r.config.WorkingDir, "initramfs.gz")
}

func (r *Runtime) getInitConfigPath() string {
	return path.Join(r.config.WorkingDir, "initconfig.json")
}
