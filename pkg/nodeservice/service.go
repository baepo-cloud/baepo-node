package nodeservice

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/pkg/chclient"
	v1pb "github.com/baepo-app/baepo-node/pkg/proto/v1"
	"github.com/baepo-app/baepo-node/pkg/proto/v1/v1connect"
	"github.com/baepo-app/baepo-node/pkg/types"
	"log/slog"
	"net"
	"net/http"
	"path"
	"sync"
	"time"
)

type Service struct {
	nodeRegistryClient v1connect.NodeRegistryServiceClient
	binaryPath         string
	socketDirectory    string
	vmLinuxPath        string
	initRamFSPath      string
	volumeGroup        string
	baseVolume         string
	lock               *sync.Mutex
	networkAllocator   *networkAllocator
	machines           map[string]*types.NodeMachine
}

func New(apiEndpoint, binaryPath, socketDirectory, vmLinuxPath, initRamFSPath, volumeGroup, baseVolume string) (*Service, error) {
	netAllocator, err := newNetworkAllocator()
	if err != nil {
		return nil, fmt.Errorf("failed to create network allocator: %w", err)
	}

	return &Service{
		nodeRegistryClient: v1connect.NewNodeRegistryServiceClient(http.DefaultClient, apiEndpoint),
		binaryPath:         binaryPath,
		socketDirectory:    socketDirectory,
		vmLinuxPath:        vmLinuxPath,
		initRamFSPath:      initRamFSPath,
		volumeGroup:        volumeGroup,
		baseVolume:         baseVolume,
		lock:               &sync.Mutex{},
		networkAllocator:   netAllocator,
		machines:           map[string]*types.NodeMachine{},
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	go func() {
		for {
			res, err := s.nodeRegistryClient.Register(context.Background(), connect.NewRequest(&v1pb.NodeRegistryRegisterRequest{
				PublicIpAddress: "localhost",
				VCpus:           4,
				Memory:          2,
				ServerEndpoint:  "http://localhost:3300",
			}))
			if err != nil {
				slog.Error("failed to register node, retrying in 5 seconds...", slog.Any("error", err))
				time.Sleep(5 * time.Second)
				continue
			}

			for res.Receive() {
			}
		}
	}()

	return nil
}

func (s *Service) newCloudHypervisorHTTPClient(machineID string) (*chclient.ClientWithResponses, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", s.getHypervisorSocketPath(machineID))
			},
		},
	}
	return chclient.NewClientWithResponses("http://localhost/api/v1", chclient.WithHTTPClient(client))
}

func (s *Service) getHypervisorSocketPath(machineID string) string {
	return path.Join(s.socketDirectory, fmt.Sprintf("%v.socket", machineID))
}

func (s *Service) getHypervisorLogPath(machineID string) string {
	return path.Join(s.socketDirectory, fmt.Sprintf("%v.log", machineID))
}

func (s *Service) getInitDaemonSocketPath(machineID string) string {
	return path.Join(s.socketDirectory, fmt.Sprintf("%v_initd.socket", machineID))
}
