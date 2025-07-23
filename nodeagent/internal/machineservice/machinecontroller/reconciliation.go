package machinecontroller

import (
	"context"
	"errors"
	"fmt"
	"github.com/sourcegraph/conc/pool"
	"time"

	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"log/slog"
)

type Reconciliation struct {
	DesiredState coretypes.MachineDesiredState
	StartedAt    time.Time
	Cancel       context.CancelFunc
}

func (c *Controller) shouldReconcile(machine *types.Machine) bool {
	switch machine.DesiredState {
	case coretypes.MachineDesiredStatePending:
		return machine.State != coretypes.MachineStatePending
	case coretypes.MachineDesiredStateRunning:
		return machine.State != coretypes.MachineStateRunning && machine.State != coretypes.MachineStateDegraded
	case coretypes.MachineDesiredStateTerminated:
		return machine.State != coretypes.MachineStateTerminated
	default:
		return false
	}
}

func (c *Controller) reconcile() {
	state := c.GetState()
	if state.Reconciliation != nil {
		c.log.Debug("already reconciling, skipping")
		return
	}

	desired := state.Machine.DesiredState
	ctx, cancel := context.WithCancel(context.Background())
	startTime := time.Now()

	_ = c.SetState(func(s *State) error {
		s.Reconciliation = &Reconciliation{
			DesiredState: desired,
			StartedAt:    startTime,
			Cancel:       cancel,
		}
		return nil
	})

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer func() {
			_ = c.SetState(func(s *State) error {
				s.Reconciliation = nil
				return nil
			})
		}()

		var (
			err      error
			newState coretypes.MachineState
		)
		switch desired {
		case coretypes.MachineDesiredStatePending:
			newState, err = c.reconcileToPending(ctx, state.Machine)
		case coretypes.MachineDesiredStateRunning:
			newState, err = c.reconcileToRunning(ctx, state.Machine)
		case coretypes.MachineDesiredStateTerminated:
			newState, err = c.reconcileToTerminated(ctx, state.Machine)
		default:
			err = fmt.Errorf("unknown desired state: %v", desired)
		}

		c.eventBus.PublishEvent(NewStateChangedMessage(newState))
		c.eventBus.PublishEvent(&ReconciliationCompleteMessage{
			Success: err == nil,
			Error:   err,
		})

		if err != nil {
			c.log.Error("reconciliation failed", slog.Any("error", err))
		} else {
			c.log.Info("reconciliation succeeded")
		}
	}()
}

func (c *Controller) reconcileToPending(ctx context.Context, machine *types.Machine) (coretypes.MachineState, error) {
	c.log.Debug("reconciling to pending state")

	if machine.RuntimePID != nil && *machine.RuntimePID > 0 {
		if machine.State != coretypes.MachineStateTerminating {
			c.eventBus.PublishEvent(NewStateChangedMessage(coretypes.MachineStateTerminating))
		}
		if err := c.terminateRuntime(ctx, machine); err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to terminate runtime: %w", err)
		}
	}

	if err := c.prepareMachine(ctx, machine); err != nil {
		return coretypes.MachineStateError, fmt.Errorf("failed to prepare resources: %w", err)
	}

	return coretypes.MachineStatePending, nil
}

func (c *Controller) reconcileToRunning(ctx context.Context, machine *types.Machine) (coretypes.MachineState, error) {
	if machine.State != coretypes.MachineStateStarting {
		c.eventBus.PublishEvent(NewStateChangedMessage(coretypes.MachineStateStarting))
	}

	c.log.Debug("reconciling to running state")
	if machine.State == coretypes.MachineStateError && machine.RuntimePID != nil && *machine.RuntimePID > 0 {
		c.log.Debug("cleaning up error state")
		if err := c.terminateRuntime(ctx, machine); err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to cleanup error state: %w", err)
		}
	}

	if err := c.prepareMachine(ctx, machine); err != nil {
		return coretypes.MachineStateError, fmt.Errorf("failed to prepare resources: %w", err)
	}

	if machine.RuntimePID == nil || *machine.RuntimePID <= 0 {
		c.log.Debug("starting runtime")
		if ctx.Err() != nil {
			return coretypes.MachineStateError, ctx.Err()
		}

		pid, err := c.runtimeProvider.Create(ctx, types.RuntimeCreateOptions{
			Machine: machine,
		})
		if err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to create runtime: %w", err)
		}

		err = c.SetState(func(state *State) error {
			state.Machine.RuntimePID = &pid
			return c.db.WithContext(ctx).Select("RuntimePID").Save(state.Machine).Error
		})
		if err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to save runtime PID: %w", err)
		}

		if err = c.runtimeProvider.Boot(ctx, machine.ID); err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to boot machine: %w", err)
		}

		c.log.Debug("runtime started successfully", slog.Int("pid", pid))
	}

	return coretypes.MachineStateRunning, nil
}

func (c *Controller) reconcileToTerminated(ctx context.Context, machine *types.Machine) (coretypes.MachineState, error) {
	c.log.Debug("reconciling to terminated state")
	if machine.RuntimePID != nil && *machine.RuntimePID > 0 {
		if machine.State != coretypes.MachineStateTerminating {
			c.eventBus.PublishEvent(NewStateChangedMessage(coretypes.MachineStateTerminating))
		}
		if err := c.terminateRuntime(ctx, machine); err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to terminate runtime: %w", err)
		}
	}

	if err := c.cleanupResources(ctx, machine); err != nil {
		return coretypes.MachineStateError, fmt.Errorf("failed to cleanup resources: %w", err)
	}

	return coretypes.MachineStateTerminated, nil
}

func (c *Controller) prepareMachine(ctx context.Context, machine *types.Machine) error {
	c.log.Debug("preparing machine", slog.Int("containers", len(machine.Containers)))
	containersByID := map[string]*types.Container{}
	for _, container := range machine.Containers {
		containersByID[container.ID] = container
	}

	p := pool.New().WithErrors().WithContext(ctx)
	p.Go(func(ctx context.Context) error {
		if err := c.prepareNetwork(ctx, machine); err != nil {
			return fmt.Errorf("failed to prepare network: %w", err)
		}

		return nil
	})

	for _, machineVolume := range machine.Volumes {
		container := containersByID[machineVolume.ContainerID]
		if container == nil {
			return fmt.Errorf("container %s not found", machineVolume.ContainerID)
		}

		p.Go(func(ctx context.Context) error {
			if machineVolume.Image != nil {
				err := c.imageProvider.Pull(ctx, machineVolume.Image)
				if err != nil {
					return fmt.Errorf("failed to pull image: %w", err)
				}
			}

			err := c.volumeProvider.Allocate(ctx, machineVolume.Volume)
			if err != nil && !errors.Is(err, types.ErrVolumeAlreadyAllocated) {
				return fmt.Errorf("failed to allocate volume (%v): %w", machineVolume.VolumeID, err)
			}

			return nil
		})
	}

	return p.Wait()
}

func (c *Controller) prepareNetwork(ctx context.Context, machine *types.Machine) error {
	if machine.NetworkInterface != nil {
		return nil
	}

	c.log.Debug("preparing network interface")
	if ctx.Err() != nil {
		return ctx.Err()
	}

	networkInterface, err := c.networkProvider.AllocateInterface(ctx)
	if err != nil {
		return fmt.Errorf("failed to allocate network interface: %w", err)
	}

	machine.NetworkInterfaceID = &networkInterface.ID
	machine.NetworkInterface = networkInterface
	return c.SetState(func(state *State) error {
		state.Machine.NetworkInterfaceID = &networkInterface.ID
		state.Machine.NetworkInterface = networkInterface
		return c.db.WithContext(ctx).Select("NetworkInterfaceID").Save(state.Machine).Error
	})
}

func (c *Controller) terminateRuntime(ctx context.Context, machine *types.Machine) error {
	c.log.Debug("terminating runtime", slog.Int("pid", *machine.RuntimePID))
	if err := c.runtimeProvider.Terminate(ctx, machine.ID); err != nil {
		return fmt.Errorf("failed to terminate runtime: %w", err)
	}

	err := c.SetState(func(state *State) error {
		state.Machine.RuntimePID = nil
		return c.db.WithContext(ctx).Select("RuntimePID").Save(state.Machine).Error
	})
	if err != nil {
		return fmt.Errorf("failed to update runtime PID: %w", err)
	}

	return nil
}

func (c *Controller) cleanupResources(ctx context.Context, machine *types.Machine) error {
	if machine.NetworkInterface != nil {
		if err := c.networkProvider.ReleaseInterface(ctx, machine.NetworkInterface.Name); err != nil {
			c.log.Error("failed to release network interface", slog.Any("error", err))
		}
	}

	for _, machineVolume := range machine.Volumes {
		if err := c.volumeProvider.Release(ctx, machineVolume.Volume); err != nil {
			c.log.Error("failed to delete volume", slog.String("volume-id", machineVolume.VolumeID), slog.Any("error", err))
		}
	}

	return nil
}
