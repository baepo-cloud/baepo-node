package initserver

import (
	"connectrpc.com/connect"
	"context"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s InitServiceServer) GetLogs(ctx context.Context, _ *connect.Request[nodev1pb.InitGetLogsRequest], stream *connect.ServerStream[nodev1pb.InitGetLogsResponse]) error {
	logChan, err := s.logService.Read(ctx)
	if err != nil {
		return err
	}

	for entry := range logChan {
		err = stream.Send(&nodev1pb.InitGetLogsResponse{
			ContainerId: entry.ContainerID,
			Error:       entry.Error,
			Content:     []byte(entry.Content),
			Timestamp:   timestamppb.New(entry.Timestamp),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
