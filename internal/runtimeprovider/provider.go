package runtimeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/chclient"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"net"
	"net/http"
	"os"
	"path"
)

type Provider struct {
	binaryPath       string
	storageDirectory string
	vmLinuxPath      string
	initRamFSPath    string
}

var _ types.RuntimeProvider = (*Provider)(nil)

func New(binaryPath, storageDirectory, vmLinuxPath, initRamFSPath string) *Provider {
	_ = os.MkdirAll(path.Join(storageDirectory, "logs"), 0644)
	_ = os.MkdirAll(path.Join(storageDirectory, "runtimes"), 0644)

	return &Provider{
		binaryPath:       binaryPath,
		storageDirectory: storageDirectory,
		vmLinuxPath:      vmLinuxPath,
		initRamFSPath:    initRamFSPath,
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
	return path.Join(p.storageDirectory, "runtimes", fmt.Sprintf("vm_%v.socket", machineID))
}

func (p *Provider) getInitDaemonSocketPath(machineID string) string {
	return path.Join(p.storageDirectory, "runtimes", fmt.Sprintf("init_%v.socket", machineID))
}

func (p *Provider) getHypervisorLogPath(machineID string) string {
	return path.Join(p.storageDirectory, "logs", fmt.Sprintf("machine_%v.log", machineID))
}
