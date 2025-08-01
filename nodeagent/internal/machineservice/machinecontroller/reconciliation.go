package machinecontroller

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"fmt"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/protobuf/types/known/emptypb"
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

	if c.isMachineRuntimeStarted(ctx, machine) {
		if machine.State != coretypes.MachineStateTerminating {
			c.eventBus.PublishEvent(NewStateChangedMessage(coretypes.MachineStateTerminating))
		}

		c.log.Debug("terminating runtime")
		if err := c.runtimeService.Terminate(ctx, machine.ID); err != nil {
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
	if machine.State == coretypes.MachineStateError && c.isMachineRuntimeStarted(ctx, machine) {
		c.log.Debug("cleaning up error state")
		if err := c.runtimeService.Terminate(ctx, machine.ID); err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to cleanup error state: %w", err)
		}
	}

	if err := c.prepareMachine(ctx, machine); err != nil {
		return coretypes.MachineStateError, fmt.Errorf("failed to prepare resources: %w", err)
	}

	if !c.isMachineRuntimeStarted(ctx, machine) {
		c.log.Debug("starting runtime")
		if ctx.Err() != nil {
			return coretypes.MachineStateError, ctx.Err()
		}

		err := c.runtimeService.Start(ctx, types.RuntimeStartOptions{Machine: machine})
		if err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to start runtime: %w", err)
		}

		c.log.Debug("runtime started successfully")
	}

	return coretypes.MachineStateRunning, nil
}

func (c *Controller) reconcileToTerminated(ctx context.Context, machine *types.Machine) (coretypes.MachineState, error) {
	c.log.Debug("reconciling to terminated state")
	if c.isMachineRuntimeStarted(ctx, machine) {
		if machine.State != coretypes.MachineStateTerminating {
			c.eventBus.PublishEvent(NewStateChangedMessage(coretypes.MachineStateTerminating))
		}
		if err := c.runtimeService.Terminate(ctx, machine.ID); err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to terminate runtime: %w", err)
		}
	}

	if err := c.networkProvider.ReleaseInterface(ctx, machine.NetworkInterface); err != nil {
		return coretypes.MachineStateError, fmt.Errorf("failed to release network interface (%v): %w", machine.NetworkInterface.ID, err)
	}

	for _, machineVolume := range machine.Volumes {
		if err := c.volumeProvider.Release(ctx, machineVolume.Volume); err != nil {
			return coretypes.MachineStateError, fmt.Errorf("failed to release volume (%v): %w", machineVolume.VolumeID, err)
		}
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
		c.log.Debug("setting up network interface")
		err := c.networkProvider.SetupInterface(ctx, machine.NetworkInterface)
		if err != nil {
			return fmt.Errorf("failed to set up network interface: %w", err)
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

func (c *Controller) isMachineRuntimeStarted(ctx context.Context, machine *types.Machine) bool {
	client, closeClient := c.runtimeService.GetClient(machine.ID)
	defer closeClient()

	_, err := client.GetState(ctx, connect.NewRequest(&emptypb.Empty{}))
	return err == nil
}
