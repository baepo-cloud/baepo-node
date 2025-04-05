package nodeservice

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	v1pb "github.com/baepo-app/baepo-oss/pkg/proto/baepo/api/v1"
	"github.com/baepo-app/baepo-oss/pkg/proto/baepo/api/v1/v1connect"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
	"google.golang.org/protobuf/types/known/emptypb"
	"log/slog"
	"net"
	"os"
	"path"
	"sync"
	"time"
)

type Service struct {
	apiClient         v1connect.NodeServiceClient
	volumeProvider    types.VolumeProvider
	networkProvider   types.NetworkProvider
	runtimeProvider   types.RuntimeProvider
	config            *types.NodeServerConfig
	authorityCert     *x509.Certificate
	tlsCert           *tls.Certificate
	lock              *sync.Mutex
	cancelRegisterCtx func()
	machines          map[string]*types.Machine
}

var _ types.NodeService = (*Service)(nil)

func New(
	apiClient v1connect.NodeServiceClient,
	volumeProvider types.VolumeProvider,
	networkProvider types.NetworkProvider,
	runtimeProvider types.RuntimeProvider,
	config *types.NodeServerConfig,
) *Service {
	return &Service{
		apiClient:       apiClient,
		volumeProvider:  volumeProvider,
		networkProvider: networkProvider,
		runtimeProvider: runtimeProvider,
		config:          config,
		lock:            &sync.Mutex{},
		machines:        map[string]*types.Machine{},
	}
}

func (s *Service) Start(ctx context.Context) error {
	slog.Info("registering node...")

	recoveredMachines, err := s.runtimeProvider.RecoverRunningMachines(ctx)
	if err != nil {
		return fmt.Errorf("failed to recover running machines: %w", err)
	}

	for _, machine := range recoveredMachines {
		if machine.NetworkInterface != nil {
			machine.NetworkInterface, err = s.networkProvider.GetInterface(machine.NetworkInterface.Name)
			if err != nil {
				slog.Info("failed to enrich machine network", slog.String("machine-id", machine.ID))
			}
		}

		s.lock.Lock()
		s.machines[machine.ID] = machine
		s.lock.Unlock()
		slog.Info("register recovered machine", slog.String("machine-id", machine.ID))
	}

	registerCtx, cancelRegisterCtx := context.WithCancel(context.Background())
	s.cancelRegisterCtx = cancelRegisterCtx
	go func() {
		for {
			select {
			case <-registerCtx.Done():
				return
			default:
				err := s.registerNode(registerCtx)
				if err != nil {
					slog.Error("failed to register node, retrying in 5 seconds...", slog.Any("error", err))
					select {
					case <-time.After(5 * time.Second):
						continue
					case <-registerCtx.Done():
						return
					}
				}
			}
		}
	}()

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.cancelRegisterCtx()
	return nil
}

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
				ApiEndpoint:     s.getEndpoint(s.config.APIAddr),
				GatewayEndpoint: s.getEndpoint(s.config.GatewayAddr),
				IpAddress:       s.config.IPAddr,
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

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-pingTicker.C:
			err = stream.Send(&v1pb.NodeConnectClientEvent{
				Event: &v1pb.NodeConnectClientEvent_Ping{
					Ping: &emptypb.Empty{},
				},
			})
			if err != nil {
				return err
			}
		}
	}
}

func (s *Service) AuthorityCertificate() *x509.Certificate {
	return s.authorityCert
}

func (s *Service) TLSCertificate() *tls.Certificate {
	return s.tlsCert
}

func (s *Service) getEndpoint(addr string) string {
	host, port, _ := net.SplitHostPort(addr)
	if host == "" {
		host = s.config.IPAddr
	}
	return net.JoinHostPort(host, port)
}

func parseCertificate(value []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(value)
	if block == nil || block.Type != "CA CERTIFICATE" {
		return nil, fmt.Errorf("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return cert, nil
}
