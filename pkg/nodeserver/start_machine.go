package nodeserver

import (
	"connectrpc.com/connect"
	"context"
	v1pb "github.com/baepo-app/baepo-node/pkg/proto/v1"
	"github.com/baepo-app/baepo-node/pkg/types"
)

func (s Server) StartMachine(ctx context.Context, req *connect.Request[v1pb.NodeStartMachineRequest]) (*connect.Response[v1pb.NodeStartMachineReply], error) {
	machine, err := s.service.StartMachine(ctx, types.NodeStartMachineOptions{
		MachineID: req.Msg.MachineId,
		Spec: types.NodeMachineSpec{
			Vcpus:  int(req.Msg.VCpus),
			Memory: req.Msg.Memory,
			Env:    req.Msg.Env,
		},
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1pb.NodeStartMachineReply{
		Machine: &v1pb.NodeMachine{
			MachineId:    machine.MachineID,
			State:        v1pb.NodeMachineState_NodeMachineState_Running,
			Pid:          int64(machine.HypervisorPID),
			TapInterface: machine.NetworkInterface.Name,
			MacAddress:   machine.NetworkInterface.MacAddress.String(),
			IpAddress:    machine.NetworkInterface.IPAddress.String(),
		},
	}), nil
}
