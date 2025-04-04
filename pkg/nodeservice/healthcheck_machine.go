package nodeservice

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-app/baepo-node/pkg/initd"
	"github.com/baepo-app/baepo-node/pkg/proto/v1/v1connect"
	"github.com/baepo-app/baepo-node/pkg/types"
	"github.com/baepo-app/baepo-node/pkg/vsock"
	"google.golang.org/protobuf/types/known/emptypb"
	"net"
	"net/http"
)

func (s *Service) HealthcheckMachine(ctx context.Context, machineID string) (*types.NodeMachine, error) {
	machine, err := s.FindMachine(ctx, machineID)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return vsock.Dial(s.getInitDaemonSocketPath(machineID), initd.ServerPort)
			},
		},
	}
	initClient := v1connect.NewInitDClient(httpClient, "http://initd")
	_, err = initClient.Healthcheck(ctx, connect.NewRequest(&emptypb.Empty{}))

	//todo set machine state
	return machine, err
}
