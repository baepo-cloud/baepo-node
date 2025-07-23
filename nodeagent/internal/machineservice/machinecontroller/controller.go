package machinecontroller

import (
	"context"
	"github.com/baepo-cloud/baepo-node/core/eventbus"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"gorm.io/gorm"
	"log/slog"
	"sync"
)

type (
	State struct {
		Machine        *types.Machine
		Reconciliation *Reconciliation
		InitListener   *InitListener
	}

	Controller struct {
		log       *slog.Logger
		state     *State
		stateLock sync.RWMutex
		eventBus  *eventbus.Bus[any]
		cancel    context.CancelFunc
		wg        sync.WaitGroup

		db              *gorm.DB
		volumeProvider  types.VolumeProvider
		networkProvider types.NetworkProvider
		runtimeProvider types.RuntimeProvider
		imageProvider   types.ImageProvider
	}
)

func New(
	machine *types.Machine,
	db *gorm.DB,
	volumeProvider types.VolumeProvider,
	networkProvider types.NetworkProvider,
	runtimeProvider types.RuntimeProvider,
	imageProvider types.ImageProvider,
) *Controller {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := &Controller{
		log: slog.With(
			slog.String("component", "machinecontroller"),
			slog.String("machine-id", machine.ID)),
		state: &State{
			Machine: machine,
		},
		eventBus:        eventbus.NewBus[any](),
		cancel:          cancel,
		db:              db,
		volumeProvider:  volumeProvider,
		networkProvider: networkProvider,
		runtimeProvider: runtimeProvider,
		imageProvider:   imageProvider,
	}

	ctrl.wg.Add(1)
	go func() {
		defer ctrl.wg.Done()
		ctrl.eventBus.StartDispatcher(ctx)
	}()

	ctrl.eventBus.SubscribeToEvents(ctrl.eventHandler)
	ctrl.eventBus.PublishEvent(&AssessStateMessage{})
	return ctrl
}

func (c *Controller) Stop() error {
	state := c.GetState()
	if state.InitListener != nil {
		state.InitListener.Cancel()
	}
	if state.Reconciliation != nil {
		state.Reconciliation.Cancel()
	}
	c.cancel()
	c.wg.Wait()
	return nil
}

func (c *Controller) GetState() *State {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()

	stateCopy := *c.state
	stateCopy.Machine = typeutil.Ptr(*c.state.Machine)
	if c.state.Reconciliation != nil {
		stateCopy.Reconciliation = typeutil.Ptr(*c.state.Reconciliation)
	}
	if c.state.InitListener != nil {
		stateCopy.InitListener = typeutil.Ptr(*c.state.InitListener)
	}
	return &stateCopy
}

func (c *Controller) SetState(updater func(state *State) error) error {
	state := c.GetState()

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	if err := updater(state); err != nil {
		return err
	}

	c.state = state
	return nil
}

func (c *Controller) SetDesiredState(desiredState coretypes.MachineDesiredState) {
	c.eventBus.PublishEvent(NewDesiredStateChangedMessage(desiredState))
}
