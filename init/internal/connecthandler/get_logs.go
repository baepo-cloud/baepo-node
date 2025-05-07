package connecthandler

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
)

func (h InitServiceHandler) GetLogs(ctx context.Context, req *connect.Request[nodev1pb.InitGetLogsRequest], stream *connect.ServerStream[nodev1pb.InitGetLogsResponse]) error {
	logChan, err := h.logService.Read(ctx, types.LogReadOptions{
		Follow: true,
	})
	if err != nil {
		return err
	}

	for entry := range logChan {
		err = stream.Send(&nodev1pb.InitGetLogsResponse{
			Fd:            entry.Fd,
			ContainerName: &entry.ContainerName,
			Content:       []byte(entry.Content),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
