package machinecontroller

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/nrednav/cuid2"
	"slices"
	"time"
)

func (c *Controller) prepareMachine(ctx context.Context) error {
	machine := c.GetMachine()
	if err := c.prepareMachineVolumes(ctx); err != nil {
		return fmt.Errorf("failed to prepare machine volume: %w", err)
	}

	if machine.NetworkInterface == nil {
		if err := c.prepareMachineNetwork(ctx); err != nil {
			return fmt.Errorf("failed to prepare machine network: %w", err)
		}
	}

	return nil
}

func (c *Controller) prepareMachineVolumes(ctx context.Context) error {
	machine := c.GetMachine()
	containerVolumes := map[string]*types.MachineVolume{}
	for _, volume := range machine.Volumes {
		containerVolumes[volume.ContainerID] = volume
	}

	for index, container := range machine.Containers {
		if _, ok := containerVolumes[container.ID]; ok {
			continue
		}

		image, err := c.imageProvider.Fetch(ctx, types.ImageFetchOptions{
			Image: container.Spec.Image,
		})
		if err != nil {
			return fmt.Errorf("failed to fetch image: %w", err)
		}

		volume, err := c.volumeProvider.Create(ctx, types.VolumeCreateOptions{
			Size:   1024, // 1 gib
			Source: image.Volume,
		})
		if err != nil {
			return fmt.Errorf("failed to create machine container volume: %w", err)
		}

		machineVolume := &types.MachineVolume{
			ID:          cuid2.Generate(),
			Position:    index,
			ContainerID: container.ID,
			Container:   container,
			MachineID:   machine.ID,
			Machine:     machine,
			ImageID:     &image.ID,
			Image:       image,
			VolumeID:    volume.ID,
			Volume:      volume,
			CreatedAt:   time.Now(),
		}
		if err = c.db.WithContext(ctx).Save(&machineVolume).Error; err != nil {
			return fmt.Errorf("failed to save machine volume: %w", err)
		}

		_ = c.updateMachine(func(machine *types.Machine) error {
			machine.Volumes = append(machine.Volumes, machineVolume)
			return nil
		})
	}

	return nil
}

func (c *Controller) prepareMachineNetwork(ctx context.Context) error {
	machine := c.GetMachine()
	if machine.NetworkInterface != nil {
		return nil
	}

	machineNetwork, err := c.networkProvider.AllocateInterface(ctx)
	if err != nil {
		return fmt.Errorf("failed to allocate machine network: %w", err)
	}

	err = c.updateMachine(func(machine *types.Machine) error {
		machine.NetworkInterfaceID = &machine.ID
		machine.NetworkInterface = machineNetwork
		return c.db.WithContext(ctx).Select("NetworkInterfaceID").Save(machine).Error
	})
	if err != nil {
		return fmt.Errorf("failed to claim network interface for a machine: %w", err)
	}

	return nil
}

func (c *Controller) terminateMachine(ctx context.Context) error {
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

	if machine.NetworkInterface != nil {
		err := c.networkProvider.ReleaseInterface(ctx, machine.NetworkInterface.Name)
		if err != nil {
			return err
		}

		_ = c.updateMachine(func(machine *types.Machine) error {
			machine.NetworkInterface = nil
			machine.NetworkInterfaceID = nil
			return c.db.WithContext(ctx).Select("NetworkInterfaceID").Save(machine).Error
		})
	}

	for index, machineVolume := range machine.Volumes {
		if err := c.volumeProvider.Delete(ctx, machineVolume.Volume); err != nil {
			return err
		}

		_ = c.updateMachine(func(machine *types.Machine) error {
			machine.Volumes = slices.Delete(machine.Volumes, index, index)
			return c.db.WithContext(ctx).Delete(&machineVolume).Error
		})
	}

	return nil
}
