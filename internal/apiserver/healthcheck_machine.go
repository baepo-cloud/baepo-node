package apiserver

import (
	"connectrpc.com/connect"
	"context"
	v1pb "github.com/baepo-app/baepo-node/internal/proto/baepo/node/v1"
)

func (s *Server) HealthcheckMachine(ctx context.Context, req *connect.Request[v1pb.NodeHealthcheckMachineRequest]) (*connect.Response[v1pb.NodeHealthcheckMachineReply], error) {
	machine, err := s.service.HealthcheckMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1pb.NodeHealthcheckMachineReply{
		Machine: &v1pb.NodeMachine{
			MachineId: machine.ID,
		},
	}), nil
}
