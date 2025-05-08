package machinecontroller

import (
	"context"
	"github.com/baepo-cloud/baepo-node/core/eventbus"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/pbadapter"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"log/slog"
	"sync"
)

type Controller struct {
	log                        *slog.Logger
	db                         *gorm.DB
	volumeProvider             types.VolumeProvider
	networkProvider            types.NetworkProvider
	runtimeProvider            types.RuntimeProvider
	imageProvider              types.ImageProvider
	machine                    *types.Machine
	initListenerMutex          sync.Mutex
	cancelInitListener         context.CancelFunc
	machineMutex               sync.RWMutex
	currentStateReconciliation *types.MachineDesiredState
	reconciliationMutex        sync.Mutex
	cancelReconciliation       func()
	eventBus                   *eventbus.Bus[*corev1pb.MachineEvent]
	eventCancelDispatcher      context.CancelFunc
}

func New(
	db *gorm.DB,
	volumeProvider types.VolumeProvider,
	networkProvider types.NetworkProvider,
	runtimeProvider types.RuntimeProvider,
	imageProvider types.ImageProvider,
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
		imageProvider:   imageProvider,
		machine:         machine,
		eventBus:        eventbus.NewBus[*corev1pb.MachineEvent](),
	}

	eventDispatcherCtx, eventCancelDispatcher := context.WithCancel(context.Background())
	ctrl.eventCancelDispatcher = eventCancelDispatcher
	go ctrl.eventBus.StartDispatcher(eventDispatcherCtx)

	ctrl.eventBus.SubscribeToEvents(ctrl.handleEvent)
	ctrl.syncInitEventsListener()

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
	c.initListenerMutex.Lock()
	defer c.initListenerMutex.Unlock()
	c.reconciliationMutex.Lock()
	defer c.reconciliationMutex.Unlock()

	if c.cancelInitListener != nil {
		c.cancelInitListener()
	}
	if c.cancelReconciliation != nil {
		c.cancelReconciliation()
	}
	if c.eventCancelDispatcher != nil {
		c.eventCancelDispatcher()
	}
}

func (c *Controller) SubscribeToEvents(handler func(ctx context.Context, event *corev1pb.MachineEvent)) func() {
	return c.eventBus.SubscribeToEvents(handler)
}

func (c *Controller) SetDesiredState(desiredState types.MachineDesiredState) {
	machineID := c.GetMachine().ID
	c.log.Info("set machine new desired state", slog.Any("desired-state", desiredState))
	c.eventBus.PublishEvent(&corev1pb.MachineEvent{
		Timestamp: timestamppb.Now(),
		MachineId: machineID,
		Event: &corev1pb.MachineEvent_DesiredStateChangedEvent{
			DesiredStateChangedEvent: &corev1pb.MachineEvent_DesiredStateChanged{
				DesiredState: pbadapter.MachineDesiredStateToProto(desiredState),
			},
		},
	})
}

func (c *Controller) updateMachine(handler func(machine *types.Machine) error) error {
	c.machineMutex.Lock()
	defer c.machineMutex.Unlock()

	return handler(c.machine)
}
