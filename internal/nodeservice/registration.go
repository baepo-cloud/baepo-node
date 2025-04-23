package nodeservice

import (
	"connectrpc.com/connect"
	"crypto/tls"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
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

	s.log.Info("node registration completed", slog.String("node-id", nodeID))
	statsTicker := time.NewTicker(30 * time.Second)
	defer statsTicker.Stop()

	if err = s.sendStatsEvent(stream); err != nil {
		return err
	}

	for {
		select {
		case <-s.ctx.Done():
			return nil
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

	err := stream.Send(&apiv1pb.NodeControllerClientEvent{
		Event: &apiv1pb.NodeControllerClientEvent_Register{
			Register: &apiv1pb.NodeControllerClientEvent_RegisterRequest{
				ClusterId:       s.config.ClusterID,
				BootstrapToken:  s.config.BootstrapToken,
				NodeToken:       nodeToken,
				IpAddress:       s.config.IPAddr,
				ApiEndpoint:     s.getEndpoint(s.config.APIAddr),
				GatewayEndpoint: s.getEndpoint(s.config.GatewayAddr),
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

	registrationResponse, ok := event.Event.(*apiv1pb.NodeControllerServerEvent_Register)
	if !ok {
		return "", fmt.Errorf("received registration response is not valid: %v", event.Event)
	}

	s.authorityCert, err = parseCertificate(registrationResponse.Register.AuthorityCert)
	if err != nil {
		return "", fmt.Errorf("failed to parse authority certificate: %w", err)
	}

	tlsCert, err := tls.X509KeyPair(registrationResponse.Register.ServerCert, registrationResponse.Register.ServerKey)
	if err != nil {
		return "", fmt.Errorf("failed to load server tls certificate: %w", err)
	}
	s.tlsCert = &tlsCert

	err = os.WriteFile(nodeTokenFilePath, []byte(registrationResponse.Register.NodeToken), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to store node token: %w", err)
	}

	return registrationResponse.Register.NodeId, nil
}

func (s *Service) sendStatsEvent(stream NodeControllerStream) error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return err
	}

	s.machineControllerLock.RLock()
	defer s.machineControllerLock.RUnlock()

	reservedMemoryMB := uint64(0)
	for _, ctrl := range s.machineControllers {
		reservedMemoryMB += ctrl.GetMachine().Spec.MemoryMB
	}

	return stream.Send(&apiv1pb.NodeControllerClientEvent{
		Event: &apiv1pb.NodeControllerClientEvent_StatsEvent{
			StatsEvent: &apiv1pb.NodeControllerClientEvent_Stats{
				TotalMemoryMb:    memInfo.Total / 1024 / 1024,
				UsedMemoryMb:     memInfo.Used / 1024 / 1024,
				ReservedMemoryMb: reservedMemoryMB,
				CpuCount:         uint32(len(cpuInfo)),
			},
		},
	})
}
