package machinecontroller

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"log/slog"
	"time"
)

func (c *Controller) startReconciliation() {
	c.reconciliationMutex.Lock()
	defer c.reconciliationMutex.Unlock()

	if c.cancelReconciliation != nil && c.reconcileToState == c.machine.DesiredState {
		return
	}

	if c.cancelReconciliation != nil {
		c.cancelReconciliation()
	}

	log := c.log.With(
		slog.String("current", string(c.machine.State)),
		slog.String("desired", string(c.machine.DesiredState)))
	startedAt := time.Now()
	reconcileCtx, cancel := context.WithCancel(context.Background())
	c.reconcileToState = c.machine.DesiredState
	c.cancelReconciliation = func() {
		cancel()
		c.cancelReconciliation = nil
	}

	go func() {
		for {
			select {
			case <-reconcileCtx.Done():
				return
			default:
				log.Info("reconciling machine state")
				err := c.reconcileState(reconcileCtx)
				log = c.log.With(slog.Duration("duration", time.Now().Sub(startedAt)))
				if err != nil {
					log.Error("failed to reconcile machine state", slog.Any("error", err))
					continue
				}

				log.Info("machine state reconciled")
				return
			}
		}
	}()
}

func (c *Controller) reconcileState(ctx context.Context) (err error) {
	// sync monitoring after reconciliation is performed
	defer c.syncMonitoring()

	switch c.machine.DesiredState {
	case types.MachineDesiredStatePending:
		return c.reconcileToPendingState(ctx)
	case types.MachineDesiredStateRunning:
		return c.reconcileToRunningState(ctx)
	case types.MachineDesiredStateTerminated:
		return c.reconcileToTerminatedState(ctx)
	default:
		return fmt.Errorf("cannot handle desired state: %v", c.machine.DesiredState)
	}
}

func (c *Controller) reconcileToPendingState(ctx context.Context) error {
	if c.machine.RuntimePID != nil && *c.machine.RuntimePID > 0 {
		err := c.runtimeProvider.Terminate(ctx, c.machine)
		if err != nil {
			return fmt.Errorf("failed to terminate machine runtime: %w", err)
		}

		c.machine.RuntimePID = nil
		if err = c.db.WithContext(ctx).Select("RuntimePID").Save(c.machine).Error; err != nil {
			return fmt.Errorf("failed to clear machine runtime pid: %w", err)
		}
	}

	return c.prepareMachine(ctx)
}

func (c *Controller) reconcileToRunningState(ctx context.Context) error {
	if err := c.prepareMachine(ctx); err != nil {
		return err
	}

	pid, err := c.runtimeProvider.Create(ctx, c.machine)
	if err != nil {
		return fmt.Errorf("failed to create machine: %w", err)
	}

	c.machine.RuntimePID = &pid
	if err = c.db.WithContext(ctx).Select("RuntimePID").Save(c.machine).Error; err != nil {
		return fmt.Errorf("failed to save machine runtime pid: %w", err)
	}

	err = c.runtimeProvider.Boot(ctx, c.machine)
	if err != nil {
		return fmt.Errorf("failed to boot machine: %w", err)
	}

	c.currentStateChan <- types.MachineStateStarting
	return nil
}

func (c *Controller) reconcileToTerminatedState(ctx context.Context) error {
	c.currentStateChan <- types.MachineStateTerminating

	if c.machine.RuntimePID != nil && *c.machine.RuntimePID > 0 {
		err := c.runtimeProvider.Terminate(ctx, c.machine)
		if err != nil {
			return fmt.Errorf("failed to terminate machine runtime: %w", err)
		}

		c.machine.RuntimePID = nil
		if err = c.db.WithContext(ctx).Select("RuntimePID").Save(c.machine).Error; err != nil {
			return fmt.Errorf("failed to clear machine runtime pid: %w", err)
		}
	}

	if c.machine.NetworkInterface != nil {
		err := c.networkProvider.ReleaseInterface(ctx, c.machine.NetworkInterface.Name)
		if err != nil {
			return err
		}
	}

	if c.machine.Volume != nil {
		err := c.volumeProvider.DeleteVolume(ctx, c.machine.Volume)
		if err != nil {
			return err
		}
	}

	c.currentStateChan <- types.MachineStateTerminated
	return nil
}
