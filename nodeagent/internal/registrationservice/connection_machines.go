package registrationservice

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"log/slog"
)

func (c *Connection) startMachineEventListener(ctx context.Context, machines []*apiv1pb.NodeControllerServerEvent_Machine) error {
	events := c.service.machineService.SubscribeToEvents(ctx)
	for _, machine := range machines {
		previousEvents, err := c.service.machineService.ListEvents(ctx, machine.MachineId)
		if err != nil {
			return fmt.Errorf("failed to list previous machine events: %w", err)
		}

		for _, event := range previousEvents {
			if err = c.sendMachineEvent(ctx, event); err != nil {
				return fmt.Errorf("failed to send machine event: %w", err)
			}
		}
	}

	go func() {
		for event := range events {
			if err := c.sendMachineEvent(ctx, event); err != nil {
				c.log.Error("failed to send machine event", slog.Any("error", err))
			}
		}
	}()

	return nil
}

func (c *Connection) sendMachineEvent(ctx context.Context, event *types.MachineEvent) error {
	anyProto, err := event.ProtoPayload()
	if err != nil {
		return err
	}

	switch proto := anyProto.(type) {
	case *corev1pb.MachineEvent:
		return c.stream.Send(&apiv1pb.NodeControllerClientEvent{
			Event: &apiv1pb.NodeControllerClientEvent_Machine{
				Machine: proto,
			},
		})
	case *corev1pb.ContainerEvent:
		return c.stream.Send(&apiv1pb.NodeControllerClientEvent{
			Event: &apiv1pb.NodeControllerClientEvent_Container{
				Container: proto,
			},
		})
	default:
		return nil
	}
}
