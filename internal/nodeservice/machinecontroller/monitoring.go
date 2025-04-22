package machinecontroller

import (
	"context"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
	"log/slog"
	"time"
)

var StatesToMonitor = []types.MachineState{
	types.MachineStateStarting,
	types.MachineStateRunning,
	types.MachineStateDegraded,
	types.MachineStateError,
}

func (c *Controller) syncMonitoring() {
	c.monitoringMutex.Lock()
	defer c.monitoringMutex.Unlock()

	machine := c.GetMachine()
	c.log.Debug("syncing monitoring", slog.Any("state", machine.State))
	if !typeutil.Includes(StatesToMonitor, machine.State) {
		if c.cancelMonitoring != nil {
			c.cancelMonitoring()
			c.cancelMonitoring = nil
		}
		return
	} else if c.cancelMonitoring != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelMonitoring = cancel
	go c.monitor(ctx, machine.ID)
}

func (c *Controller) monitor(ctx context.Context, machineID string) {
	consecutiveError := 0
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.log.Debug("performing machine healthcheck")
			checkCtx, cancelCheckCtx := context.WithTimeout(ctx, time.Second)
			err := c.runtimeProvider.Healthcheck(checkCtx, machineID)
			cancelCheckCtx()

			if err != nil {
				c.log.Warn("machine healthcheck failed", slog.Any("error", err))

				consecutiveError++
				if consecutiveError >= 3 {
					c.updateCurrentState(types.MachineStateError)
				} else if consecutiveError > 0 {
					c.updateCurrentState(types.MachineStateDegraded)
				}
			} else {
				consecutiveError = 0
				c.updateCurrentState(types.MachineStateRunning)
			}
		}
	}
}
