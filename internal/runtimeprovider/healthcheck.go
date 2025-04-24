package runtimeprovider

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-cloud/baepo-node/internal/initd"
	"github.com/baepo-cloud/baepo-node/internal/vsock"
	"github.com/baepo-cloud/baepo-proto/go/baepo/initd/v1/initdv1pbconnect"
	"google.golang.org/protobuf/types/known/emptypb"
	"net"
	"net/http"
)

func (p *Provider) Healthcheck(ctx context.Context, machineID string) error {
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return vsock.Dial(p.getInitDaemonSocketPath(machineID), initd.ServerPort)
			},
		},
	}
	initClient := initdv1pbconnect.NewInitDClient(httpClient, "http://initd")
	_, err := initClient.Healthcheck(ctx, connect.NewRequest(&emptypb.Empty{}))
	return err
}
