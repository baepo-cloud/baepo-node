package machineservice

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/machineservice/machinecontroller"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"gorm.io/gorm"
	"log/slog"
	"sync"
)

type Service struct {
	log                   *slog.Logger
	db                    *gorm.DB
	volumeProvider        types.VolumeProvider
	networkProvider       types.NetworkProvider
	runtimeProvider       types.RuntimeProvider
	imageProvider         types.ImageProvider
	config                *types.Config
	authorityCert         *x509.Certificate
	tlsCert               *tls.Certificate
	gcWorkerCtx           context.Context
	cancelGCWorker        context.CancelFunc
	machineControllerLock sync.RWMutex
	machineControllers    map[string]*machinecontroller.Controller
	machineEvents         chan *corev1pb.MachineEvent
}

var _ types.MachineService = (*Service)(nil)

func New(
	db *gorm.DB,
	volumeProvider types.VolumeProvider,
	networkProvider types.NetworkProvider,
	runtimeProvider types.RuntimeProvider,
	imageProvider types.ImageProvider,
	config *types.Config,
) *Service {
	return &Service{
		log:                   slog.With(slog.String("component", "machineservice")),
		db:                    db,
		volumeProvider:        volumeProvider,
		networkProvider:       networkProvider,
		runtimeProvider:       runtimeProvider,
		imageProvider:         imageProvider,
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

	s.gcWorkerCtx, s.cancelGCWorker = context.WithCancel(context.Background())
	go s.startGCWorker()
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if s.cancelGCWorker != nil {
		s.cancelGCWorker()
	}
	return nil
}

func (s *Service) loadMachines(ctx context.Context) error {
	s.log.Info("loading machines")

	var machines []*types.Machine
	err := s.db.WithContext(ctx).
		Preload("Volumes.Volume").
		Preload("Volumes.Image.Volume").
		Preload("Containers").
		Preload("NetworkInterface").
		Where("machines.state NOT IN ?", []types.MachineState{types.MachineStateTerminated}).
		Find(&machines).
		Error
	if err != nil {
		return fmt.Errorf("failed to retrieve machines: %w", err)
	}

	s.machineControllerLock.Lock()
	defer s.machineControllerLock.Unlock()
	for _, machine := range machines {
		s.machineControllers[machine.ID] = s.newMachineController(machine)
	}
	return nil
}

func (s *Service) newMachineController(machine *types.Machine) *machinecontroller.Controller {
	ctrl := machinecontroller.New(
		s.db, s.volumeProvider, s.networkProvider, s.runtimeProvider, s.imageProvider,
		machine,
	)
	ctrl.SubscribeToEvents(func(ctx context.Context, payload any) {
		go func() {
			if machineEvent, ok := payload.(*corev1pb.MachineEvent); ok {
				s.machineEvents <- machineEvent
			}
		}()
	})
	return ctrl
}
