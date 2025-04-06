package apiserver

import (
	"connectrpc.com/connect"
	"context"
	v1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Server) StopMachine(ctx context.Context, req *connect.Request[v1pb.NodeStopMachineRequest]) (*connect.Response[emptypb.Empty], error) {
	_, err := s.service.StopMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}
