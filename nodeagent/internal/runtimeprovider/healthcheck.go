package runtimeprovider

import (
	"connectrpc.com/connect"
	"context"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/vsock"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"google.golang.org/protobuf/types/known/emptypb"
	"net"
	"net/http"
)

func (p *Provider) Healthcheck(ctx context.Context, machineID string) error {
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return vsock.Dial(p.getInitDaemonSocketPath(machineID), coretypes.InitServerPort)
			},
		},
	}
	initClient := nodev1pbconnect.NewInitClient(httpClient, "http://init")
	_, err := initClient.Healthcheck(ctx, connect.NewRequest(&emptypb.Empty{}))
	return err
}
