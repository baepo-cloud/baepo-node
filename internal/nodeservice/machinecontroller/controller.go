package machinecontroller

import (
	"context"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"gorm.io/gorm"
	"log/slog"
	"sync"
)

type (
	Controller struct {
		log                  *slog.Logger
		db                   *gorm.DB
		volumeProvider       types.VolumeProvider
		networkProvider      types.NetworkProvider
		runtimeProvider      types.RuntimeProvider
		machine              *types.Machine
		cancelWatch          context.CancelFunc
		cancelMonitoring     context.CancelFunc
		monitoringMutex      sync.Mutex
		desiredStateChan     chan types.MachineDesiredState
		currentStateChan     chan types.MachineState
		machineMutex         sync.RWMutex
		reconcileToState     types.MachineDesiredState
		reconciliationMutex  sync.Mutex
		cancelReconciliation func()
		watcher              func(machine *types.Machine)
	}
)

func New(
	db *gorm.DB,
	volumeProvider types.VolumeProvider,
	networkProvider types.NetworkProvider,
	runtimeProvider types.RuntimeProvider,
	machine *types.Machine,
	watcher func(machine *types.Machine),
) *Controller {
	ctrl := &Controller{
		log: slog.With(
			slog.String("component", "machinecontroller"),
			slog.String("machine-id", machine.ID)),
		db:               db,
		volumeProvider:   volumeProvider,
		networkProvider:  networkProvider,
		runtimeProvider:  runtimeProvider,
		machine:          machine,
		desiredStateChan: make(chan types.MachineDesiredState),
		currentStateChan: make(chan types.MachineState),
		watcher:          watcher,
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
			err := c.updateMachine(func(machine *types.Machine) error {
				machine.DesiredState = desiredState
				return c.db.WithContext(ctx).Select("DesiredState").Save(machine).Error
			})
			if err != nil {
				c.log.Error("failed to update machine desired state",
					slog.String("desired-state", string(desiredState)),
					slog.Any("error", err))
			}

			machine := c.GetMachine()
			if c.watcher != nil {
				c.watcher(machine)
			}
			
			if !machine.State.MatchDesiredState(machine.DesiredState) {
				go c.startReconciliation()
			}
		case state := <-c.currentStateChan:
			err := c.updateMachine(func(machine *types.Machine) error {
				machine.State = state
				return c.db.WithContext(ctx).Select("State").Save(machine).Error
			})
			if err != nil {
				c.log.Error("failed to update machine state",
					slog.String("state", string(state)),
					slog.Any("error", err))
			}

			machine := c.GetMachine()
			if c.watcher != nil {
				c.watcher(machine)
			}

			if machine.State == types.MachineStateTerminated {
				c.Stop()
				return
			} else if !machine.State.MatchDesiredState(machine.DesiredState) {
				go c.startReconciliation()
			}
		}
	}
}

func (c *Controller) GetMachine() *types.Machine {
	c.machineMutex.RLock()
	defer c.machineMutex.RUnlock()

	copiedMachine := *c.machine
	return &copiedMachine
}

func (c *Controller) UpdateDesiredState(desiredState types.MachineDesiredState) {
	machine := c.GetMachine()
	if machine.DesiredState != desiredState {
		c.desiredStateChan <- desiredState
	}
}

func (c *Controller) updateCurrentState(state types.MachineState) {
	machine := c.GetMachine()
	if machine.State != state {
		c.currentStateChan <- state
	}
}

func (c *Controller) Stop() {
	c.monitoringMutex.Lock()
	defer c.monitoringMutex.Unlock()
	c.reconciliationMutex.Lock()
	defer c.reconciliationMutex.Unlock()

	c.cancelWatch()
	if c.cancelMonitoring != nil {
		c.cancelMonitoring()
	}
	if c.cancelReconciliation != nil {
		c.cancelReconciliation()
	}
}

func (c *Controller) updateMachine(handler func(machine *types.Machine) error) error {
	c.machineMutex.Lock()
	defer c.machineMutex.Unlock()
	return handler(c.machine)
}
