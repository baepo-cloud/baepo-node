package machinecontroller

import (
	"context"
	"github.com/baepo-cloud/baepo-node/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"gorm.io/gorm"
	"log/slog"
	"sync"
)

type (
	Controller struct {
		log                             *slog.Logger
		db                              *gorm.DB
		volumeProvider                  types.VolumeProvider
		networkProvider                 types.NetworkProvider
		runtimeProvider                 types.RuntimeProvider
		machine                         *types.Machine
		cancelMonitoring                context.CancelFunc
		monitoringMutex                 sync.Mutex
		monitoringConsecutiveErrorCount int
		machineMutex                    sync.RWMutex
		reconcileToState                types.MachineDesiredState
		reconciliationMutex             sync.Mutex
		cancelReconciliation            func()
		eventsChan                      chan *corev1pb.MachineEvent
		eventCancelDispatcher           context.CancelFunc
		eventHandlers                   map[string]func(context.Context, *corev1pb.MachineEvent)
		eventHandlersLock               sync.RWMutex
	}
)

func New(
	db *gorm.DB,
	volumeProvider types.VolumeProvider,
	networkProvider types.NetworkProvider,
	runtimeProvider types.RuntimeProvider,
	machine *types.Machine,
) *Controller {
	ctrl := &Controller{
		log: slog.With(
			slog.String("component", "machinecontroller"),
			slog.String("machine-id", machine.ID)),
		db:              db,
		volumeProvider:  volumeProvider,
		networkProvider: networkProvider,
		runtimeProvider: runtimeProvider,
		machine:         machine,
		eventsChan:      make(chan *corev1pb.MachineEvent),
		eventHandlers:   make(map[string]func(context.Context, *corev1pb.MachineEvent)),
	}

	eventDispatcherCtx, eventCancelDispatcher := context.WithCancel(context.Background())
	ctrl.eventCancelDispatcher = eventCancelDispatcher
	go ctrl.startEventDispatcher(eventDispatcherCtx)

	ctrl.SubscribeToEvents(ctrl.handleEvent)
	ctrl.syncMonitoring()

	if !machine.State.MatchDesiredState(machine.DesiredState) {
		go ctrl.startReconciliation()
	}

	return ctrl
}

func (c *Controller) GetMachine() *types.Machine {
	c.machineMutex.RLock()
	defer c.machineMutex.RUnlock()

	copiedMachine := *c.machine
	return &copiedMachine
}

func (c *Controller) Stop() {
	c.monitoringMutex.Lock()
	defer c.monitoringMutex.Unlock()
	c.reconciliationMutex.Lock()
	defer c.reconciliationMutex.Unlock()

	if c.cancelMonitoring != nil {
		c.cancelMonitoring()
	}
	if c.cancelReconciliation != nil {
		c.cancelReconciliation()
	}
	if c.eventCancelDispatcher != nil {
		c.eventCancelDispatcher()
	}
}

func (c *Controller) updateMachine(handler func(machine *types.Machine) error) error {
	c.machineMutex.Lock()
	defer c.machineMutex.Unlock()
	return handler(c.machine)
}
