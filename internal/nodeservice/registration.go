package nodeservice

import (
	"connectrpc.com/connect"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/apiserver/v1pbadapter"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"log/slog"
	"os"
	"path"
	"time"
)

type NodeControllerStream = *connect.BidiStreamForClient[apiv1pb.NodeControllerClientEvent, apiv1pb.NodeControllerServerEvent]

func (s *Service) startRegistrationWorker() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			err := s.connectNodeToController()
			if err != nil {
				s.log.Error("failed to register node, retrying in 5 seconds", slog.Any("error", err))
				select {
				case <-time.After(5 * time.Second):
					continue
				case <-s.ctx.Done():
					return
				}
			}
		}
	}
}

func (s *Service) connectNodeToController() error {
	s.log.Info("starting node registration")
	stream := s.apiClient.Events(s.ctx)

	nodeID, err := s.sendRegisterEvent(stream)
	if err != nil {
		return err
	}

	serverEvents := make(chan *apiv1pb.NodeControllerServerEvent)
	go func() {
		defer close(serverEvents)

		for {
			req, err := stream.Receive()
			if errors.Is(err, io.EOF) {
				break
			} else if err == nil {
				serverEvents <- req
			}
		}
	}()

	s.log.Info("node registration completed", slog.String("node-id", nodeID))
	statsTicker := time.NewTicker(10 * time.Second)
	defer statsTicker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return nil
		case event := <-s.events:
			if err = stream.Send(event); err != nil {
				return err
			}
		case event := <-serverEvents:
			if err = s.processServerEvent(event); err != nil {
				return err
			}
		case <-statsTicker.C:
			if err = s.sendStatsEvent(stream); err != nil {
				return err
			}
		}
	}
}

func (s *Service) sendRegisterEvent(stream NodeControllerStream) (string, error) {
	nodeTokenFilePath := path.Join(s.config.StorageDirectory, "token")

	var nodeToken *string
	if b, err := os.ReadFile(nodeTokenFilePath); err == nil {
		nodeToken = typeutil.Ptr(string(b))
	}

	stats, err := s.newStatsEvent()
	if err != nil {
		return "", err
	}

	err = stream.Send(&apiv1pb.NodeControllerClientEvent{
		Event: &apiv1pb.NodeControllerClientEvent_RegisterEvent{
			RegisterEvent: &apiv1pb.NodeControllerClientEvent_Register{
				ClusterId:       s.config.ClusterID,
				BootstrapToken:  s.config.BootstrapToken,
				NodeToken:       nodeToken,
				IpAddress:       s.config.IPAddr,
				ApiEndpoint:     s.getEndpoint(s.config.APIAddr),
				GatewayEndpoint: s.getEndpoint(s.config.GatewayAddr),
				Stats:           stats,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to send registration request: %w", err)
	}

	event, err := stream.Receive()
	if err != nil {
		return "", fmt.Errorf("failed to receive registration response: %w", err)
	}

	registrationCompleted, ok := event.Event.(*apiv1pb.NodeControllerServerEvent_RegistrationCompletedEvent)
	if !ok {
		return "", fmt.Errorf("received registration response is not valid: %v", event.Event)
	}

	s.authorityCert, err = parseCertificate(registrationCompleted.RegistrationCompletedEvent.AuthorityCert)
	if err != nil {
		return "", fmt.Errorf("failed to parse authority certificate: %w", err)
	}

	tlsCert, err := tls.X509KeyPair(
		registrationCompleted.RegistrationCompletedEvent.ServerCert,
		registrationCompleted.RegistrationCompletedEvent.ServerKey,
	)
	if err != nil {
		return "", fmt.Errorf("failed to load server tls certificate: %w", err)
	}
	s.tlsCert = &tlsCert

	err = os.WriteFile(nodeTokenFilePath, []byte(registrationCompleted.RegistrationCompletedEvent.NodeToken), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to store node token: %w", err)
	}

	err = s.processExpectedMachines(registrationCompleted.RegistrationCompletedEvent.ExpectedMachines)
	if err != nil {
		return "", fmt.Errorf("failed to process expected machine list: %w", err)
	}

	return registrationCompleted.RegistrationCompletedEvent.NodeId, nil
}

func (s *Service) sendStatsEvent(stream NodeControllerStream) error {
	stats, err := s.newStatsEvent()
	if err != nil {
		return err
	}

	return stream.Send(&apiv1pb.NodeControllerClientEvent{
		Event: &apiv1pb.NodeControllerClientEvent_StatsEvent{
			StatsEvent: stats,
		},
	})
}

func (s *Service) newStatsEvent() (*apiv1pb.NodeControllerClientEvent_Stats, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, err
	}

	s.machineControllerLock.RLock()
	defer s.machineControllerLock.RUnlock()

	reservedMemoryMB := uint64(0)
	for _, ctrl := range s.machineControllers {
		reservedMemoryMB += ctrl.GetMachine().Spec.MemoryMB
	}

	return &apiv1pb.NodeControllerClientEvent_Stats{
		TotalMemoryMb:    memInfo.Total / 1024 / 1024,
		UsedMemoryMb:     memInfo.Used / 1024 / 1024,
		ReservedMemoryMb: reservedMemoryMB,
		CpuCount:         uint32(len(cpuInfo)),
	}, nil
}

func (s *Service) processExpectedMachines(machines []*apiv1pb.NodeControllerServerEvent_MachineSpec) error {
	s.log.Info("processing expected machines list")
	expectedMachines := map[string]bool{}
	for _, spec := range machines {
		expectedMachines[spec.MachineId] = true
		if err := s.reconcileWithExpectedMachine(spec); err != nil {
			return fmt.Errorf("failed to reconcile machine: %w", err)
		}
	}

	var machinesToTerminate []string
	s.machineControllerLock.RLock()
	for machineID := range s.machineControllers {
		if _, ok := expectedMachines[machineID]; !ok {
			machinesToTerminate = append(machinesToTerminate, machineID)
		}
	}
	s.machineControllerLock.RUnlock()

	for _, machineID := range machinesToTerminate {
		_, err := s.UpdateMachineDesiredState(s.ctx, types.NodeUpdateMachineDesiredStateOptions{
			MachineID:    machineID,
			DesiredState: types.MachineDesiredStateTerminated,
		})
		if err != nil {
			return fmt.Errorf("failed to terminate machine: %w", err)
		}
	}

	return nil
}

func (s *Service) reconcileWithExpectedMachine(spec *apiv1pb.NodeControllerServerEvent_MachineSpec) error {
	desiredState := v1pbadapter.ProtoToMachineDesiredState(spec.DesiredState)
	log := s.log.With(slog.String("machine-id", spec.MachineId), slog.Any("desired-state", desiredState))

	s.machineControllerLock.RLock()
	ctrl, ok := s.machineControllers[spec.MachineId]
	s.machineControllerLock.RUnlock()

	if !ok {
		log.Info("missing machine, creating")
		_, err := s.CreateMachine(s.ctx, types.NodeCreateMachineOptions{
			MachineID:    spec.MachineId,
			DesiredState: desiredState,
			Spec:         v1pbadapter.ProtoToMachineSpec(spec.Spec),
		})
		if err != nil {
			return fmt.Errorf("failed to create machine: %w", err)
		}

		return nil
	}

	if current := ctrl.GetMachine().DesiredState; current != desiredState {
		log.Info("desired state mismatch, updating", slog.Any("current-desired-state", current))
		_, err := s.UpdateMachineDesiredState(s.ctx, types.NodeUpdateMachineDesiredStateOptions{
			MachineID:    spec.MachineId,
			DesiredState: desiredState,
		})
		if err != nil {
			return fmt.Errorf("failed to update machine desired state: %w", err)
		}

		return nil
	}

	return nil
}

func (s *Service) processServerEvent(unknownEvent *apiv1pb.NodeControllerServerEvent) error {
	switch event := unknownEvent.Event.(type) {
	case *apiv1pb.NodeControllerServerEvent_CreateMachineEvent:
		_, err := s.CreateMachine(s.ctx, types.NodeCreateMachineOptions{
			MachineID:    event.CreateMachineEvent.MachineId,
			DesiredState: v1pbadapter.ProtoToMachineDesiredState(event.CreateMachineEvent.DesiredState),
			Spec:         v1pbadapter.ProtoToMachineSpec(event.CreateMachineEvent.Spec),
		})
		return err
	case *apiv1pb.NodeControllerServerEvent_UpdateMachineDesiredStateEvent:
		_, err := s.UpdateMachineDesiredState(s.ctx, types.NodeUpdateMachineDesiredStateOptions{
			MachineID:    event.UpdateMachineDesiredStateEvent.MachineId,
			DesiredState: v1pbadapter.ProtoToMachineDesiredState(event.UpdateMachineDesiredStateEvent.DesiredState),
		})
		return err
	case *apiv1pb.NodeControllerServerEvent_PingEvent:
		return nil
	default:
		return errors.New("unknown event")
	}
}

func (s *Service) newMachineEvent(machine *types.Machine) *apiv1pb.NodeControllerClientEvent_MachineEvent {
	return &apiv1pb.NodeControllerClientEvent_MachineEvent{
		MachineEvent: &nodev1pb.MachineEvent{
			MachineId:          machine.ID,
			State:              v1pbadapter.MachineStateToProto(machine.State),
			DesiredState:       v1pbadapter.MachineDesiredStateToProto(machine.DesiredState),
			StartedAt:          nil,
			ExpiresAt:          nil,
			TerminatedAt:       nil,
			TerminationCause:   nil,
			TerminationDetails: nil,
			Timestamp:          timestamppb.Now(),
		},
	}
}
