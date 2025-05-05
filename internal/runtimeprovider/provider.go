package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/runtimeprovider/chclient"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
)

type Provider struct {
	config  *types.NodeServerConfig
	gcMutex sync.RWMutex
}

var _ types.RuntimeProvider = (*Provider)(nil)

func New(config *types.NodeServerConfig) *Provider {
	_ = os.MkdirAll(path.Join(config.StorageDirectory, "logs"), 0644)
	_ = os.MkdirAll(path.Join(config.StorageDirectory, "runtimes"), 0644)
	return &Provider{config: config}
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
