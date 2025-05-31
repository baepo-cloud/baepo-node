package initserver

import (
	"connectrpc.com/connect"
	"context"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s InitServiceServer) Events(ctx context.Context, _ *connect.Request[emptypb.Empty], stream *connect.ServerStream[nodev1pb.InitEventsResponse]) error {
	for event := range s.containerService.Events(ctx) {
		proto := adaptContainerEventToProto(event)
		if proto == nil {
			continue
		} else if err := stream.Send(proto); err != nil {
			return err
		}
	}

	return nil
}

func adaptContainerEventToProto(event any) *nodev1pb.InitEventsResponse {
	switch value := event.(type) {
	case *types.ContainerStateChangedEvent:
		proto := &nodev1pb.InitEventsResponse_ContainerStateChangedEvent{
			ContainerId:  value.ContainerID,
			ExitCode:     value.ExitCode,
			Healthy:      value.Healthy,
			RestartCount: value.RestartCount,
		}
		if value.StartedAt != nil {
			proto.StartedAt = timestamppb.New(*value.StartedAt)
		}
		if value.ExitedAt != nil {
			proto.ExitedAt = timestamppb.New(*value.ExitedAt)
		}
		if value.ExitError != nil {
			proto.ExitError = typeutil.Ptr(value.ExitError.Error())
		}
		if value.HealthcheckError != nil {
			proto.HealthcheckError = typeutil.Ptr(value.HealthcheckError.Error())
		}

		switch {
		case value.ExitedAt != nil:
			proto.State = corev1pb.ContainerState_MachineContainerState_Exited
		case value.StartedAt != nil:
			proto.State = corev1pb.ContainerState_MachineContainerState_Running
		default:
			proto.State = corev1pb.ContainerState_MachineContainerState_Unknown
		}

		return &nodev1pb.InitEventsResponse{
			EventId:   value.EventID,
			Timestamp: timestamppb.New(value.Timestamp),
			Event: &nodev1pb.InitEventsResponse_ContainerStateChanged{
				ContainerStateChanged: proto,
			},
		}
	default:
		return nil
	}
}
