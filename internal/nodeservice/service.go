package nodeservice

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/nodeservice/machinecontroller"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/api/v1/apiv1pbconnect"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"gorm.io/gorm"
	"log/slog"
	"net"
	"sync"
)

type Service struct {
	log                   *slog.Logger
	db                    *gorm.DB
	apiClient             apiv1pbconnect.NodeControllerServiceClient
	volumeProvider        types.VolumeProvider
	networkProvider       types.NetworkProvider
	runtimeProvider       types.RuntimeProvider
	config                *types.NodeServerConfig
	authorityCert         *x509.Certificate
	tlsCert               *tls.Certificate
	ctx                   context.Context
	cancelCtx             context.CancelFunc
	machineControllerLock sync.RWMutex
	machineControllers    map[string]*machinecontroller.Controller
	machineEvents         chan *corev1pb.MachineEvent
}

var _ types.NodeService = (*Service)(nil)

func New(
	db *gorm.DB,
	apiClient apiv1pbconnect.NodeControllerServiceClient,
	volumeProvider types.VolumeProvider,
	networkProvider types.NetworkProvider,
	runtimeProvider types.RuntimeProvider,
	config *types.NodeServerConfig,
) *Service {
	return &Service{
		log:                   slog.With(slog.String("component", "nodeservice")),
		db:                    db,
		apiClient:             apiClient,
		volumeProvider:        volumeProvider,
		networkProvider:       networkProvider,
		runtimeProvider:       runtimeProvider,
		config:                config,
		machineControllerLock: sync.RWMutex{},
		machineControllers:    map[string]*machinecontroller.Controller{},
		machineEvents:         make(chan *corev1pb.MachineEvent),
	}
}

func (s *Service) Start(ctx context.Context) error {
	if err := s.loadMachines(ctx); err != nil {
		return fmt.Errorf("failed to load machines: %w", err)
	}

	s.ctx, s.cancelCtx = context.WithCancel(context.Background())
	go s.startGCWorker()
	go s.startRegistrationWorker()
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.cancelCtx()
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
