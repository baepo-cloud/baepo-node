package registrationservice

import (
	"connectrpc.com/connect"
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	apiv1pb "github.com/baepo-cloud/baepo-proto/go/baepo/api/v1"
	"github.com/baepo-cloud/baepo-proto/go/baepo/api/v1/apiv1pbconnect"
	"gorm.io/gorm"
	"log/slog"
)

type (
	Service struct {
		log            *slog.Logger
		db             *gorm.DB
		apiClient      apiv1pbconnect.NodeControllerServiceClient
		config         *types.Config
		machineService types.MachineService
		authorityCert  *x509.Certificate
		tlsCert        *tls.Certificate
		workerCtx      context.Context
		cancelWorker   context.CancelFunc
	}

	NodeControllerStream = *connect.BidiStreamForClient[apiv1pb.NodeControllerClientEvent, apiv1pb.NodeControllerServerEvent]
)

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
	s.workerCtx, s.cancelWorker = context.WithCancel(context.Background())
	go s.startRegistrationWorker()
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if s.cancelWorker != nil {
		s.cancelWorker()
	}
	return nil
}
