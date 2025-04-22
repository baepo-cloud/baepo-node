package machinecontroller

import (
	"context"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"gorm.io/gorm"
	"log/slog"
	"sync"
)

type Controller struct {
	log              *slog.Logger
	db               *gorm.DB
	volumeProvider   types.VolumeProvider
	networkProvider  types.NetworkProvider
	runtimeProvider  types.RuntimeProvider
	machine          *types.Machine
	cancelWatch      context.CancelFunc
	cancelMonitoring context.CancelFunc
	desiredStateChan chan types.MachineDesiredState
	currentStateChan chan types.MachineState

	reconcileToState     types.MachineDesiredState
	reconciliationMutex  sync.Mutex
	cancelReconciliation func()
}

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
		db:                  db,
		volumeProvider:      volumeProvider,
		networkProvider:     networkProvider,
		runtimeProvider:     runtimeProvider,
		machine:             machine,
		desiredStateChan:    make(chan types.MachineDesiredState),
		currentStateChan:    make(chan types.MachineState),
		reconciliationMutex: sync.Mutex{},
	}

	watchCtx, cancelWatch := context.WithCancel(context.Background())
	ctrl.cancelWatch = cancelWatch
	go ctrl.watchStateChanges(watchCtx)

	return ctrl
}

func (c *Controller) watchStateChanges(ctx context.Context) {
	if !c.machine.State.MatchDesiredState(c.machine.DesiredState) {
		go c.startReconciliation()
	}
	
	c.syncMonitoring()

	for {
		select {
		case <-ctx.Done():
			return
		case desiredState := <-c.desiredStateChan:
			c.machine.DesiredState = desiredState
			if err := c.db.WithContext(ctx).Select("DesiredState").Save(c.machine).Error; err != nil {
				c.log.Error("failed to update machine desired state",
					slog.String("desired-state", string(desiredState)),
					slog.Any("error", err))
			}
			if !c.machine.State.MatchDesiredState(c.machine.DesiredState) {
				go c.startReconciliation()
			}
		case state := <-c.currentStateChan:
			c.machine.State = state
			if err := c.db.WithContext(ctx).Select("State").Save(c.machine).Error; err != nil {
				c.log.Error("failed to update machine state",
					slog.String("state", string(state)),
					slog.Any("error", err))
			} else if c.machine.State == types.MachineStateTerminated {
				c.Stop()
				return
			}
			if !c.machine.State.MatchDesiredState(c.machine.DesiredState) {
				go c.startReconciliation()
			}
		}
	}
}

func (c *Controller) GetMachine() *types.Machine {
	return c.machine
}

func (c *Controller) UpdateDesiredState(desiredState types.MachineDesiredState) {
	c.desiredStateChan <- desiredState
}

func (c *Controller) Stop() {
	c.cancelWatch()
	if c.cancelMonitoring != nil {
		c.cancelMonitoring()
	}
}
