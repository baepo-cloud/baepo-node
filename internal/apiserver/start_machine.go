package apiserver

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-cloud/baepo-node/internal/types"
	v1pb "github.com/baepo-cloud/baepo-node/pkg/proto/baepo/node/v1"
)

func (s *Server) StartMachine(ctx context.Context, req *connect.Request[v1pb.NodeStartMachineRequest]) (*connect.Response[v1pb.NodeStartMachineReply], error) {
	machine, err := s.service.StartMachine(ctx, types.NodeStartMachineOptions{
		MachineID: req.Msg.MachineId,
		Spec: types.MachineSpec{
			Vcpus:    int(req.Msg.VCpus),
			MemoryMB: req.Msg.MemoryMb,
			Env:      req.Msg.Env,
		},
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1pb.NodeStartMachineReply{
		Machine: &v1pb.NodeMachine{
			MachineId:    machine.ID,
			State:        v1pb.NodeMachineState_NodeMachineState_Running,
			Pid:          int64(machine.RuntimePID),
			TapInterface: machine.NetworkInterface.Name,
			MacAddress:   machine.NetworkInterface.MacAddress.String(),
			IpAddress:    machine.NetworkInterface.IPAddress.String(),
		},
	}), nil
}
