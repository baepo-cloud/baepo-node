package apiserver

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-cloud/baepo-node/internal/pbadapter"
	"github.com/baepo-cloud/baepo-node/internal/types"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
)

func (s *Server) ListMachines(ctx context.Context, _ *connect.Request[nodev1pb.NodeListMachinesRequest]) (*connect.Response[nodev1pb.NodeListMachinesResponse], error) {
	machines, err := s.service.ListMachines(ctx)
	if err != nil {
		return nil, err
	}

	res := &nodev1pb.NodeListMachinesResponse{
		Machines: make([]*nodev1pb.Machine, len(machines)),
	}
	for index, machine := range machines {
		res.Machines[index] = s.adaptMachine(machine)
	}
	return connect.NewResponse(res), nil
}

func (s *Server) GetMachine(ctx context.Context, req *connect.Request[nodev1pb.NodeGetMachineRequest]) (*connect.Response[nodev1pb.NodeGetMachineResponse], error) {
	machine, err := s.service.FindMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&nodev1pb.NodeGetMachineResponse{
		Machine: s.adaptMachine(machine),
	}), nil
}

func (s *Server) adaptMachine(machine *types.Machine) *nodev1pb.Machine {
	return &nodev1pb.Machine{
		MachineId:    machine.ID,
		State:        pbadapter.MachineStateToProto(machine.State),
		DesiredState: pbadapter.MachineDesiredStateToProto(machine.DesiredState),
	}
}
