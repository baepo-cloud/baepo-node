package runtimeprovider

import (
	"context"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/vsock"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/runtimeprovider/chclient"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"net"
	"net/http"
	"path"
	"sync"
	"time"
)

type Provider struct {
	config  *types.Config
	gcMutex sync.RWMutex
}

var _ types.RuntimeProvider = (*Provider)(nil)

func New(config *types.Config) *Provider {
	return &Provider{config: config}
}

func (p *Provider) NewInitClient(machineID string) (nodev1pbconnect.InitClient, func()) {
	var conns []net.Conn
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				conn, err := vsock.DialContext(ctx, p.getInitDaemonSocketPath(machineID), coretypes.InitServerPort)
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

func (p *Provider) newCloudHypervisorHTTPClient(machineID string) (*chclient.ClientWithResponses, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", p.getHypervisorSocketPath(machineID))
			},
		},
	}
	return chclient.NewClientWithResponses("http://localhost/api/v1", chclient.WithHTTPClient(client))
}

func (p *Provider) getMachineDir(machineID string) string {
	return path.Join(p.config.StorageDirectory, "machines", machineID)
}

func (p *Provider) getHypervisorSocketPath(machineID string) string {
	return path.Join(p.getMachineDir(machineID), "vmm.socket")
}

func (p *Provider) getInitDaemonSocketPath(machineID string) string {
	return path.Join(p.getMachineDir(machineID), "init.socket")
}

func (p *Provider) getHypervisorLogPath(machineID string) string {
	return path.Join(p.getMachineDir(machineID), "machine.log")
}

func (p *Provider) getInitRamFSPath(machineID string) string {
	return path.Join(p.getMachineDir(machineID), "initramfs.gz")
}

func (p *Provider) getInitConfigPath(machineID string) string {
	return path.Join(p.getMachineDir(machineID), "initconfig.json")
}
