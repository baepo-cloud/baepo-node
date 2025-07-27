package runtimeservice

import (
	"connectrpc.com/connect"
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Service) Terminate(ctx context.Context, machineID string) error {
	client, closeClient := s.GetClient(machineID)
	defer closeClient()

	_, err := client.Terminate(ctx, connect.NewRequest(&emptypb.Empty{}))
	return err
}
