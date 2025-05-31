package registrationservice

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"io"
	"log/slog"
	"os"
	"path"
	"time"
)

func (s *Service) startRegistrationWorker() {
	for {
		select {
		case <-s.workerCtx.Done():
			return
		default:
			err := s.connectNodeToController()
			if err != nil {
				s.log.Error("failed to register node, retrying in 5 seconds", slog.Any("error", err))
				select {
				case <-time.After(5 * time.Second):
					continue
				case <-s.workerCtx.Done():
					return
				}
			}
		}
	}
}

func (s *Service) connectNodeToController() error {
	s.log.Info("starting node registration")
	stream := s.apiClient.Events(s.workerCtx)

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
		case <-s.workerCtx.Done():
			return nil
		//case event := <-s.machineEvents:
		//	err = stream.Send(&apiv1pb.NodeControllerClientEvent{
		//		Event: &apiv1pb.NodeControllerClientEvent_Machine{
		//			Machine: event,
		//		},
		//	})
		//	if err != nil {
		//		return err
		//	}
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
		Event: &apiv1pb.NodeControllerClientEvent_Register_{
			Register: &apiv1pb.NodeControllerClientEvent_Register{
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

	registrationCompleted, ok := event.Event.(*apiv1pb.NodeControllerServerEvent_RegistrationCompleted)
	if !ok {
		return "", fmt.Errorf("received registration response is not valid: %v", event.Event)
	}

	s.authorityCert, err = parseCertificate(registrationCompleted.RegistrationCompleted.AuthorityCert)
	if err != nil {
		return "", fmt.Errorf("failed to parse authority certificate: %w", err)
	}

	tlsCert, err := tls.X509KeyPair(
		registrationCompleted.RegistrationCompleted.ServerCert,
		registrationCompleted.RegistrationCompleted.ServerKey,
	)
	if err != nil {
		return "", fmt.Errorf("failed to load server tls certificate: %w", err)
	}
	s.tlsCert = &tlsCert

	err = os.WriteFile(nodeTokenFilePath, []byte(registrationCompleted.RegistrationCompleted.NodeToken), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to store node token: %w", err)
	}

	err = s.processExpectedMachines(registrationCompleted.RegistrationCompleted.ExpectedMachines)
	if err != nil {
		return "", fmt.Errorf("failed to process expected machine list: %w", err)
	}

	return registrationCompleted.RegistrationCompleted.NodeId, nil
}

func (s *Service) sendStatsEvent(stream NodeControllerStream) error {
	stats, err := s.newStatsEvent()
	if err != nil {
		return err
	}

	return stream.Send(&apiv1pb.NodeControllerClientEvent{
		Event: &apiv1pb.NodeControllerClientEvent_Stats_{
			Stats: stats,
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

	machines, err := s.machineService.List(s.workerCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to list machines: %w", err)
	}

	reservedMemoryMB := uint64(0)
	for _, machine := range machines {
		reservedMemoryMB += machine.Spec.MemoryMB
	}

	return &apiv1pb.NodeControllerClientEvent_Stats{
		TotalMemoryMb:    memInfo.Total / 1024 / 1024,
		UsedMemoryMb:     memInfo.Used / 1024 / 1024,
		ReservedMemoryMb: reservedMemoryMB,
		CpuCount:         uint32(len(cpuInfo)),
	}, nil
}

func (s *Service) processExpectedMachines(machines []*apiv1pb.NodeControllerServerEvent_Machine) error {
	s.log.Info("processing expected machines list")
	expectedMachines := map[string]bool{}
	for _, spec := range machines {
		expectedMachines[spec.MachineId] = true
		if err := s.reconcileWithExpectedMachine(spec); err != nil {
			return fmt.Errorf("failed to reconcile machine: %w", err)
		}
	}

	currentMachines, err := s.machineService.List(s.workerCtx)
	if err != nil {
		return fmt.Errorf("failed to list machines: %w", err)
	}

	for _, machine := range currentMachines {
		if _, ok := expectedMachines[machine.ID]; ok {
			continue
		}

		_, err = s.machineService.UpdateDesiredState(s.workerCtx, types.MachineUpdateDesiredStateOptions{
			MachineID:    machine.ID,
			DesiredState: types.MachineDesiredStateTerminated,
		})
		if err != nil {
			return fmt.Errorf("failed to terminate machine: %w", err)
		}
	}
	return nil
}

func (s *Service) reconcileWithExpectedMachine(spec *apiv1pb.NodeControllerServerEvent_Machine) error {
	desiredState := pbadapter.ProtoToMachineDesiredState(spec.DesiredState)
	log := s.log.With(slog.String("machine-id", spec.MachineId), slog.Any("desired-state", desiredState))
	machine, err := s.machineService.FindByID(s.workerCtx, spec.MachineId)
	if errors.Is(err, types.ErrMachineNotFound) {
		log.Info("missing machine, creating")
		if err = s.createMachine(s.workerCtx, spec); err != nil {
			return fmt.Errorf("failed to create machine: %w", err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("failed to find machine: %w", err)
	} else if current := machine.DesiredState; current != desiredState {
		log.Info("desired state mismatch, updating", slog.Any("current-desired-state", current))
		_, err = s.machineService.UpdateDesiredState(s.workerCtx, types.MachineUpdateDesiredStateOptions{
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
	case *apiv1pb.NodeControllerServerEvent_CreateMachine:
		return s.createMachine(s.workerCtx, event.CreateMachine)
	case *apiv1pb.NodeControllerServerEvent_UpdateMachineDesiredState:
		_, err := s.machineService.UpdateDesiredState(s.workerCtx, types.MachineUpdateDesiredStateOptions{
			MachineID:    event.UpdateMachineDesiredState.MachineId,
			DesiredState: pbadapter.ProtoToMachineDesiredState(event.UpdateMachineDesiredState.DesiredState),
		})
		return err
	case *apiv1pb.NodeControllerServerEvent_Ping:
		return nil
	default:
		return errors.New("unknown event")
	}
}

func (s *Service) createMachine(ctx context.Context, pbMachine *apiv1pb.NodeControllerServerEvent_Machine) error {
	opts := types.MachineCreateOptions{
		MachineID:    pbMachine.MachineId,
		DesiredState: pbadapter.ProtoToMachineDesiredState(pbMachine.DesiredState),
		Spec:         pbadapter.ProtoToMachineSpec(pbMachine.Spec),
		Containers:   make([]types.MachineCreateContainerOptions, len(pbMachine.Containers)),
	}
	for index, container := range pbMachine.Containers {
		opts.Containers[index] = types.MachineCreateContainerOptions{
			ContainerID: container.ContainerId,
			Spec:        pbadapter.ProtoToContainerSpec(container.Spec),
		}
	}

	_, err := s.machineService.Create(ctx, opts)
	return err
}
