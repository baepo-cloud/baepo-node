package machinecontroller

import (
	"context"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"log/slog"
	"time"
)

func (c *Controller) startReconciliation() {
	c.reconciliationMutex.Lock()
	defer c.reconciliationMutex.Unlock()

	machine := c.GetMachine()
	if c.cancelReconciliation != nil && c.currentStateReconciliation != nil &&
		*c.currentStateReconciliation == machine.DesiredState {
		return
	}

	if c.cancelReconciliation != nil {
		c.cancelReconciliation()
	}

	log := c.log.With(
		slog.String("current", string(machine.State)),
		slog.String("desired", string(machine.DesiredState)))
	startedAt := time.Now()
	reconcileCtx, cancel := context.WithCancel(context.Background())
	c.currentStateReconciliation = &machine.DesiredState
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
				err := c.reconcileState(reconcileCtx, machine.DesiredState)
				log = c.log.With(slog.Duration("duration", time.Now().Sub(startedAt)))
				if err != nil {
					log.Error("failed to reconcile machine state", slog.Any("error", err))
					continue
				}

				c.currentStateReconciliation = nil
				log.Info("machine state reconciled")
				return
			}
		}
	}()
}

func (c *Controller) reconcileState(ctx context.Context, desiredState coretypes.MachineDesiredState) error {
	switch desiredState {
	case coretypes.MachineDesiredStatePending:
		return c.reconcileToPendingState(ctx)
	case coretypes.MachineDesiredStateRunning:
		return c.reconcileToRunningState(ctx)
	case coretypes.MachineDesiredStateTerminated:
		return c.reconcileToTerminatedState(ctx)
	default:
		return fmt.Errorf("cannot handle desired state: %v", c.machine.DesiredState)
	}
}

func (c *Controller) reconcileToPendingState(ctx context.Context) error {
	machine := c.GetMachine()
	if machine.RuntimePID != nil && *machine.RuntimePID > 0 {
		err := c.runtimeProvider.Terminate(ctx, machine.ID)
		if err != nil {
			return fmt.Errorf("failed to terminate machine runtime: %w", err)
		}

		err = c.updateMachine(func(machine *types.Machine) error {
			machine.RuntimePID = nil
			return c.db.WithContext(ctx).Select("RuntimePID").Save(machine).Error
		})
		if err != nil {
			return fmt.Errorf("failed to clear machine runtime pid: %w", err)
		}
	}

	return c.prepareMachine(ctx)
}

func (c *Controller) reconcileToRunningState(ctx context.Context) error {
	machine := c.GetMachine()
	if machine.State == coretypes.MachineStateError {
		if err := c.terminateMachine(ctx); err != nil {
			return fmt.Errorf("failed to terminate machine: %w", err)
		}
	}

	if err := c.prepareMachine(ctx); err != nil {
		return err
	}

	machine = c.GetMachine()
	pid, err := c.runtimeProvider.Create(ctx, types.RuntimeCreateOptions{
		Machine: machine,
	})
	if err != nil {
		return fmt.Errorf("failed to create machine: %w", err)
	}

	err = c.updateMachine(func(machine *types.Machine) error {
		machine.RuntimePID = &pid
		return c.db.WithContext(ctx).Select("RuntimePID").Save(machine).Error
	})
	if err != nil {
		return fmt.Errorf("failed to save machine runtime pid: %w", err)
	}

	err = c.runtimeProvider.Boot(ctx, machine.ID)
	if err != nil {
		return fmt.Errorf("failed to boot machine: %w", err)
	}

	c.dispatchMachineStateChangeEvent(coretypes.MachineStateStarting)
	return nil
}

func (c *Controller) reconcileToTerminatedState(ctx context.Context) error {
	c.dispatchMachineStateChangeEvent(coretypes.MachineStateTerminating)
	if err := c.terminateMachine(ctx); err != nil {
		return fmt.Errorf("failed to terminate machine: %w", err)
	}
	c.dispatchMachineStateChangeEvent(coretypes.MachineStateTerminated)
	return nil
}
