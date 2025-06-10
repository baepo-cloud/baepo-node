package machineservice

import (
	"context"
	"fmt"
	"github.com/alphadose/haxmap"
	"github.com/baepo-cloud/baepo-node/core/eventbus"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/machineservice/machinecontroller"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"gorm.io/gorm"
	"log/slog"
)

type Service struct {
	log                   *slog.Logger
	db                    *gorm.DB
	volumeProvider        types.VolumeProvider
	networkProvider       types.NetworkProvider
	runtimeProvider       types.RuntimeProvider
	imageProvider         types.ImageProvider
	config                *types.Config
	cancelGCWorker        context.CancelFunc
	machineControllers    *haxmap.Map[string, *machinecontroller.Controller]
	cancelEventDispatcher context.CancelFunc
	machineEvents         *eventbus.Bus[*types.MachineEvent]
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
		log:                slog.With(slog.String("component", "machineservice")),
		db:                 db,
		volumeProvider:     volumeProvider,
		networkProvider:    networkProvider,
		runtimeProvider:    runtimeProvider,
		imageProvider:      imageProvider,
		config:             config,
		machineControllers: haxmap.New[string, *machinecontroller.Controller](),
		machineEvents:      eventbus.NewBus[*types.MachineEvent](),
	}
}

func (s *Service) Start(ctx context.Context) error {
	eventDispatcherCtx, cancelEventDispatcher := context.WithCancel(context.Background())
	go s.machineEvents.StartDispatcher(eventDispatcherCtx)
	s.cancelEventDispatcher = cancelEventDispatcher

	if err := s.loadMachines(ctx); err != nil {
		return fmt.Errorf("failed to load machines: %w", err)
	}

	gcWorkerCtx, cancelGCWorker := context.WithCancel(context.Background())
	go s.startGCWorker(gcWorkerCtx)
	s.cancelGCWorker = cancelGCWorker

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if s.cancelGCWorker != nil {
		s.cancelGCWorker()
	}
	if s.cancelEventDispatcher != nil {
		s.cancelEventDispatcher()
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
		Where("machines.state NOT IN ?", []coretypes.MachineState{coretypes.MachineStateTerminated}).
		Find(&machines).
		Error
	if err != nil {
		return fmt.Errorf("failed to retrieve machines: %w", err)
	}

	for _, machine := range machines {
		s.machineControllers.Set(machine.ID, s.newMachineController(machine))
	}
	return nil
}

func (s *Service) newMachineController(machine *types.Machine) *machinecontroller.Controller {
	ctrl := machinecontroller.New(s.db, s.volumeProvider, s.networkProvider, s.runtimeProvider,
		s.imageProvider, machine)
	ctrl.SubscribeToEvents(s.newMachineEventsHandler(machine))
	return ctrl
}
