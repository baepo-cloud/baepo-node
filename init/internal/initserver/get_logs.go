package initserver

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s InitServiceServer) GetLogs(ctx context.Context, req *connect.Request[nodev1pb.InitGetLogsRequest], stream *connect.ServerStream[nodev1pb.InitGetLogsResponse]) error {
	logChan, err := s.logService.Read(ctx, types.LogReadOptions{
		Follow: req.Msg.Follow,
	})
	if err != nil {
		return err
	}

	for entry := range logChan {
		if ctr := req.Msg.Container; ctr != nil && !(*ctr == entry.ContainerID || *ctr == entry.ContainerName) {
			continue
		}

		err = stream.Send(&nodev1pb.InitGetLogsResponse{
			ContainerId:   entry.ContainerID,
			ContainerName: entry.ContainerName,
			Error:         entry.Error,
			Content:       []byte(entry.Content),
			Timestamp:     timestamppb.New(entry.Timestamp),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
