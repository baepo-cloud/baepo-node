package machinecontroller

import (
	"context"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"log/slog"
	"time"
)

func (c *Controller) syncMonitoring() {
	if c.machine.State != types.MachineStateStarting && c.machine.State != types.MachineStateRunning {
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
	go c.monitor(ctx)
}

func (c *Controller) monitor(ctx context.Context) {
	consecutiveError := 0
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.log.Debug("performing machine healthcheck")
			if err := c.runtimeProvider.Healthcheck(ctx, c.machine); err != nil {
				c.log.Warn("machine healthcheck failed", slog.Any("error", err))
				consecutiveError++

				if consecutiveError >= 3 {
					c.currentStateChan <- types.MachineStateError
				} else if consecutiveError > 0 {
					c.currentStateChan <- types.MachineStateDegraded
				}
			} else {
				consecutiveError = 0
				c.currentStateChan <- types.MachineStateRunning
			}
		}
	}
}
