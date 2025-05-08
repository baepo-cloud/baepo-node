package runtimeprovider

import (
	"context"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/vsock"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/runtimeprovider/chclient"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
)

type Provider struct {
	config  *types.Config
	gcMutex sync.RWMutex
}

var _ types.RuntimeProvider = (*Provider)(nil)

func New(config *types.Config) *Provider {
	_ = os.MkdirAll(path.Join(config.StorageDirectory, "logs"), 0644)
	_ = os.MkdirAll(path.Join(config.StorageDirectory, "runtimes"), 0644)
	return &Provider{config: config}
}

func (p *Provider) NewInitClient(machineID string) (nodev1pbconnect.InitClient, func()) {
	conn, err := vsock.Dial(p.getInitDaemonSocketPath(machineID), coretypes.InitServerPort)
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return conn, err
			},
		},
	}
	return nodev1pbconnect.NewInitClient(httpClient, "http://init"), func() {
		if conn != nil {
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

func (p *Provider) getHypervisorSocketPath(machineID string) string {
	return path.Join(p.config.StorageDirectory, "runtimes", fmt.Sprintf("%v_vm.socket", machineID))
}

func (p *Provider) getInitDaemonSocketPath(machineID string) string {
	return path.Join(p.config.StorageDirectory, "runtimes", fmt.Sprintf("%v_init.socket", machineID))
}

func (p *Provider) getHypervisorLogPath(machineID string) string {
	return path.Join(p.config.StorageDirectory, "logs", fmt.Sprintf("machine_%v.log", machineID))
}

func (p *Provider) getInitRamFSPath(machineID string) string {
	return filepath.Join(p.config.StorageDirectory, "runtimes", fmt.Sprintf("%v_initramfs.gz", machineID))
}
