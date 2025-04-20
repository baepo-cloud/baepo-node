package nodeservice

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/api/v1/v1connect"
	"gorm.io/gorm"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Service struct {
	db                *gorm.DB
	apiClient         v1connect.NodeControllerServiceClient
	volumeProvider    types.VolumeProvider
	networkProvider   types.NetworkProvider
	runtimeProvider   types.RuntimeProvider
	config            *types.NodeServerConfig
	authorityCert     *x509.Certificate
	tlsCert           *tls.Certificate
	lock              *sync.Mutex
	cancelRegisterCtx func()
}

var _ types.NodeService = (*Service)(nil)

func New(
	db *gorm.DB,
	apiClient v1connect.NodeControllerServiceClient,
	volumeProvider types.VolumeProvider,
	networkProvider types.NetworkProvider,
	runtimeProvider types.RuntimeProvider,
	config *types.NodeServerConfig,
) *Service {
	return &Service{
		db:              db,
		apiClient:       apiClient,
		volumeProvider:  volumeProvider,
		networkProvider: networkProvider,
		runtimeProvider: runtimeProvider,
		config:          config,
	}
}

func (s *Service) Start(ctx context.Context) error {
	slog.Info("registering node...")

	var machines []*types.Machine
	err := s.db.WithContext(ctx).
		Joins("Volume").
		Joins("NetworkInterface").
		Where("machines.status NOT IN ?", []types.MachineStatus{types.MachineStatusTerminated}).
		Find(&machines).
		Error
	if err != nil {
		return fmt.Errorf("failed to retrieve machines")
	}

	for _, machine := range machines {
		if machine.Status == types.MachineStatusTerminating {
			if _, err = s.StopMachine(ctx, machine.ID); err != nil {
				return fmt.Errorf("failed to stop machine: %w", err)
			}
			continue
		}

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
