package apiserver

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
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

func (s *Server) GetMachineLogs(ctx context.Context, req *connect.Request[nodev1pb.NodeGetMachineLogsRequest], writeStream *connect.ServerStream[nodev1pb.NodeGetMachineLogsResponse]) error {
	machine, err := s.service.FindMachine(ctx, req.Msg.MachineId)
	if err != nil {
		return err
	}

	client, closeClient := s.runtimeProvider.NewInitClient(machine.ID)
	defer closeClient()

	readStream, err := client.GetLogs(ctx, connect.NewRequest(&nodev1pb.InitGetLogsRequest{
		Container: req.Msg.ContainerName,
		Follow:    req.Msg.Follow,
	}))
	if err != nil {
		return err
	}

	for readStream.Receive() {
		msg := readStream.Msg()
		err = writeStream.Send(&nodev1pb.NodeGetMachineLogsResponse{
			Error:         msg.Error,
			ContainerName: msg.ContainerName,
			Content:       msg.Content,
			Timestamp:     msg.Timestamp,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) adaptMachine(machine *types.Machine) *nodev1pb.Machine {
	return &nodev1pb.Machine{
		MachineId:    machine.ID,
		State:        pbadapter.MachineStateToProto(machine.State),
		DesiredState: pbadapter.MachineDesiredStateToProto(machine.DesiredState),
	}
}
