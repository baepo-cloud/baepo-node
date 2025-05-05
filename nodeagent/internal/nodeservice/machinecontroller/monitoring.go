package machinecontroller

import (
	"context"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	corev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/core/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	ticker := time.NewTicker(5 * time.Second)
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

			healthcheckEvent := &corev1pb.MachineEvent_Healthcheck{}
			if err != nil {
				healthcheckEvent.Error = typeutil.Ptr(err.Error())
			}

			c.PublishEvent(&corev1pb.MachineEvent{
				Timestamp: timestamppb.Now(),
				MachineId: machineID,
				Event: &corev1pb.MachineEvent_HealthcheckEvent{
					HealthcheckEvent: healthcheckEvent,
				},
			})
		}
	}
}
