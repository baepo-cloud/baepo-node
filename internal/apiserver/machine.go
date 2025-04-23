package apiserver

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-cloud/baepo-node/internal/apiserver/v1pbadapter"
	"github.com/baepo-cloud/baepo-node/internal/types"
	v1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
)

func (s *Server) ListMachines(ctx context.Context, _ *connect.Request[v1pb.NodeListMachinesRequest]) (*connect.Response[v1pb.NodeListMachinesResponse], error) {
	machines, err := s.service.ListMachines(ctx)
	if err != nil {
		return nil, err
	}

	res := &v1pb.NodeListMachinesResponse{
		Machines: make([]*v1pb.Machine, len(machines)),
	}
	for index, machine := range machines {
		res.Machines[index] = s.adaptMachine(machine)
	}
	return connect.NewResponse(res), nil
}

func (s *Server) CreateMachine(ctx context.Context, req *connect.Request[v1pb.NodeCreateMachineRequest]) (*connect.Response[v1pb.NodeCreateMachineResponse], error) {
	machine, err := s.service.CreateMachine(ctx, types.NodeCreateMachineOptions{
		MachineID:    req.Msg.MachineId,
		DesiredState: v1pbadapter.ProtoToMachineDesiredState(req.Msg.DesiredState),
		Spec: types.MachineSpec{
			Cpus:     req.Msg.Spec.Cpus,
			MemoryMB: req.Msg.Spec.MemoryMb,
			Env:      req.Msg.Spec.Containers[0].Env,
			Image:    req.Msg.Spec.Containers[0].Image,
		},
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1pb.NodeCreateMachineResponse{
		Machine: s.adaptMachine(machine),
	}), nil
}

func (s *Server) GetMachine(ctx context.Context, req *connect.Request[v1pb.NodeGetMachineRequest]) (*connect.Response[v1pb.NodeGetMachineResponse], error) {
	machine, err := s.service.FindMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1pb.NodeGetMachineResponse{
		Machine: s.adaptMachine(machine),
	}), nil
}

func (s *Server) UpdateMachineDesiredState(ctx context.Context, req *connect.Request[v1pb.NodeUpdateMachineDesiredStateRequest]) (*connect.Response[v1pb.NodeUpdateMachineDesiredStateResponse], error) {
	machine, err := s.service.UpdateMachineDesiredState(ctx, types.NodeUpdateMachineDesiredStateOptions{
		MachineID:    req.Msg.MachineId,
		DesiredState: v1pbadapter.ProtoToMachineDesiredState(req.Msg.DesiredState),
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1pb.NodeUpdateMachineDesiredStateResponse{
		Machine: s.adaptMachine(machine),
	}), nil
}

func (s *Server) adaptMachine(machine *types.Machine) *v1pb.Machine {
	return &v1pb.Machine{
		MachineId:    machine.ID,
		State:        v1pbadapter.MachineStateToProto(machine.State),
		DesiredState: v1pbadapter.MachineDesiredStateToProto(machine.DesiredState),
	}
}
