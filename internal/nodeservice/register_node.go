package nodeservice

import (
	"connectrpc.com/connect"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
	v1pb "github.com/baepo-cloud/baepo-oss/pkg/proto/baepo/api/v1"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"log/slog"
	"os"
	"path"
	"time"
)

func (s *Service) registerNode(ctx context.Context) error {
	slog.Info("starting node registration...")
	stream := s.apiClient.Connect(ctx)
	nodeTokenFilePath := path.Join(s.config.StorageDirectory, "token")

	var nodeToken *string
	if b, err := os.ReadFile(nodeTokenFilePath); err == nil {
		nodeToken = typeutil.Ptr(string(b))
	}

	err := stream.Send(&v1pb.NodeConnectClientEvent{
		Event: &v1pb.NodeConnectClientEvent_Register{
			Register: &v1pb.NodeConnectClientEvent_RegisterRequest{
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
		return fmt.Errorf("failed to send registration request: %w", err)
	}

	event, err := stream.Receive()
	if err != nil {
		return fmt.Errorf("failed to receive registration response: %w", err)
	}

	registrationResponse, ok := event.Event.(*v1pb.NodeConnectServerEvent_Register)
	if !ok {
		return fmt.Errorf("received registration response is not valid: %v", event.Event)
	}

	s.authorityCert, err = parseCertificate(registrationResponse.Register.AuthorityCert)
	if err != nil {
		return fmt.Errorf("failed to parse authority certificate: %w", err)
	}

	tlsCert, err := tls.X509KeyPair(registrationResponse.Register.ServerCert, registrationResponse.Register.ServerKey)
	if err != nil {
		return fmt.Errorf("failed to load server tls certificate: %w", err)
	}
	s.tlsCert = &tlsCert

	err = os.WriteFile(nodeTokenFilePath, []byte(registrationResponse.Register.NodeToken), 0644)
	if err != nil {
		return fmt.Errorf("failed to store node token: %w", err)
	}

	slog.Info("node registration completed", slog.String("node-id", registrationResponse.Register.NodeId))

	statsTicker := time.NewTicker(30 * time.Second)
	defer statsTicker.Stop()

	if err = s.sendStatsEvent(stream); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-statsTicker.C:
			if err = s.sendStatsEvent(stream); err != nil {
				return err
			}
		}
	}
}

func (s *Service) sendStatsEvent(stream *connect.BidiStreamForClient[v1pb.NodeConnectClientEvent, v1pb.NodeConnectServerEvent]) error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return err
	}

	var runningMachineIDs []string
	reservedMemory := uint64(0)
	for _, machine := range s.machines {
		runningMachineIDs = append(runningMachineIDs, machine.ID)
		reservedMemory += uint64(machine.Spec.MemoryMB)
	}

	return stream.Send(&v1pb.NodeConnectClientEvent{
		Event: &v1pb.NodeConnectClientEvent_Stats{
			Stats: &v1pb.NodeConnectClientEvent_StatsEvent{
				TotalMemory:       memInfo.Total,
				UsedMemory:        memInfo.Used,
				CpuCount:          uint32(len(cpuInfo)),
				RunningMachineIds: runningMachineIDs,
				ReservedMemory:    reservedMemory,
			},
		},
	})
}
