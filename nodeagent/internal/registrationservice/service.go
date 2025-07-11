package registrationservice

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/api/v1/apiv1pbconnect"
	"gorm.io/gorm"
	"log/slog"
	"net"
)

type Service struct {
	log            *slog.Logger
	db             *gorm.DB
	apiClient      apiv1pbconnect.NodeControllerServiceClient
	config         *types.Config
	machineService types.MachineService
	authorityCert  *x509.Certificate
	tlsCert        *tls.Certificate
	cancelWorker   context.CancelFunc
}

var _ types.RegistrationService = (*Service)(nil)

func New(
	db *gorm.DB,
	apiClient apiv1pbconnect.NodeControllerServiceClient,
	config *types.Config,
	machineService types.MachineService,
) *Service {
	return &Service{
		log:            slog.With(slog.String("component", "registrationservice")),
		db:             db,
		apiClient:      apiClient,
		config:         config,
		machineService: machineService,
	}
}

func (s *Service) Start(ctx context.Context) error {
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	go s.startRegistrationWorker(workerCtx)
	s.cancelWorker = cancelWorker

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if s.cancelWorker != nil {
		s.cancelWorker()
	}
	return nil
}

func (s *Service) getEndpoint(addr string) string {
	host, port, _ := net.SplitHostPort(addr)
	if host == "" {
		host = s.config.IPAddr
	}
	return net.JoinHostPort(host, port)
}
