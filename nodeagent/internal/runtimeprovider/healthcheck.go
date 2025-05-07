package runtimeprovider

import (
	"connectrpc.com/connect"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (p *Provider) Healthcheck(ctx context.Context, machineID string) error {
	_, err := p.NewInitClient(machineID).Healthcheck(ctx, connect.NewRequest(&emptypb.Empty{}))
	return err
}
