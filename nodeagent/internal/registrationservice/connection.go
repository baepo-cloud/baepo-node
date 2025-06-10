package registrationservice

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/v1pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	"io"
	"log/slog"
	"time"
)

type (
	NodeControllerStream = *connect.BidiStreamForClient[apiv1pb.NodeControllerClientEvent, apiv1pb.NodeControllerServerEvent]

	Connection struct {
		service *Service
		log     *slog.Logger
		stream  NodeControllerStream
		nodeID  string
	}
)

func (s *Service) openConnection(ctx context.Context) error {
	conn := &Connection{
		service: s,
		log:     s.log.With(slog.String("component", "registrationconnection")),
		stream:  s.apiClient.Events(ctx),
	}
	conn.log.Info("starting node registration")

	registration, err := conn.sendRegistrationEvent(ctx)
	if err != nil {
		return fmt.Errorf("failed to send registration event: %w", err)
	}

	conn.nodeID = registration.NodeId
	conn.log = s.log.With(slog.String("node-id", registration.NodeId))
	conn.log.Info("node registration completed")

	if err = conn.startMachineEventListener(ctx, registration.ExpectedMachines); err != nil {
		return fmt.Errorf("failed to start machine event listener: %w", err)
	} else if err = s.syncMachines(ctx, registration.ExpectedMachines); err != nil {
		return fmt.Errorf("failed to sync machines: %w", err)
	}
	return conn.dispatchLoop(ctx)
}

func (c *Connection) dispatchLoop(ctx context.Context) error {
	statsTicker := time.NewTicker(10 * time.Second)
	defer statsTicker.Stop()

	incomingEvents := make(chan *apiv1pb.NodeControllerServerEvent)
	go func() {
		defer close(incomingEvents)

		for {
			req, err := c.stream.Receive()
			if errors.Is(err, io.EOF) {
				break
			} else if err == nil {
				incomingEvents <- req
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-incomingEvents:
			if err := c.handleIncomingEvent(ctx, event); err != nil {
				return fmt.Errorf("failed to handle incoming event: %w", err)
			}
		case <-statsTicker.C:
			if err := c.sendStatsEvent(ctx); err != nil {
				return fmt.Errorf("failed to send stats event: %w", err)
			}
		}
	}
}

func (c *Connection) handleIncomingEvent(ctx context.Context, anyEvent *apiv1pb.NodeControllerServerEvent) error {
	switch event := anyEvent.Event.(type) {
	case *apiv1pb.NodeControllerServerEvent_CreateMachine:
		return c.service.createMachine(ctx, event.CreateMachine)
	case *apiv1pb.NodeControllerServerEvent_UpdateMachineDesiredState:
		_, err := c.service.machineService.UpdateDesiredState(ctx, types.MachineUpdateDesiredStateOptions{
			MachineID:    event.UpdateMachineDesiredState.MachineId,
			DesiredState: v1pbadapter.ToMachineDesiredState(event.UpdateMachineDesiredState.DesiredState),
		})
		return err
	case *apiv1pb.NodeControllerServerEvent_Ping:
		return nil
	default:
		return errors.New("unknown event")
	}
}
