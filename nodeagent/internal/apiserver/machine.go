package apiserver

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-cloud/baepo-node/core/v1pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) ListMachines(ctx context.Context, _ *connect.Request[nodev1pb.NodeListMachinesRequest]) (*connect.Response[nodev1pb.NodeListMachinesResponse], error) {
	machines, err := s.machineService.List(ctx)
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
	machine, err := s.machineService.FindByID(ctx, req.Msg.MachineId)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&nodev1pb.NodeGetMachineResponse{
		Machine: s.adaptMachine(machine),
	}), nil
}

func (s *Server) GetMachineLogs(ctx context.Context, req *connect.Request[nodev1pb.NodeGetMachineLogsRequest], stream *connect.ServerStream[nodev1pb.NodeGetMachineLogsResponse]) error {
	logs, err := s.machineService.GetMachineLogs(ctx, types.MachineGetMachineLogsOptions{
		MachineID: req.Msg.MachineId,
		Follow:    req.Msg.Follow,
	})
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case log, ok := <-logs:
			if !ok {
				return nil
			}

			err = stream.Send(&nodev1pb.NodeGetMachineLogsResponse{
				Content: log.Content,
			})
			if err != nil {
				return err
			}
		}
	}
}

func (s *Server) GetContainerLogs(ctx context.Context, req *connect.Request[nodev1pb.NodeGetContainerLogsRequest], stream *connect.ServerStream[nodev1pb.NodeGetContainerLogsResponse]) error {
	logs, err := s.machineService.GetContainerLogs(ctx, types.MachineGetContainerLogsOptions{
		MachineID:   req.Msg.MachineId,
		ContainerID: req.Msg.ContainerId,
		Follow:      req.Msg.Follow,
	})
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case log, ok := <-logs:
			if !ok {
				return nil
			}

			err = stream.Send(&nodev1pb.NodeGetContainerLogsResponse{
				ContainerId: log.ContainerID,
				Error:       log.Error,
				Content:     log.Content,
				Timestamp:   timestamppb.New(log.Timestamp),
			})
			if err != nil {
				return err
			}
		}
	}
}

func (s *Server) adaptMachine(machine *types.Machine) *nodev1pb.Machine {
	return &nodev1pb.Machine{
		MachineId:    machine.ID,
		State:        v1pbadapter.FromMachineState(machine.State),
		DesiredState: v1pbadapter.FromMachineDesiredState(machine.DesiredState),
	}
}
