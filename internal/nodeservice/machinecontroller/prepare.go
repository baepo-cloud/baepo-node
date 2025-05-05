package machinecontroller

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/nrednav/cuid2"
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
	for index, ctr := range machine.Spec.Containers {
		image, err := c.imageProvider.Fetch(ctx, types.ImageFetchOptions{
			Image: ctr.Image,
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
			ID:        cuid2.Generate(),
			Position:  index,
			Container: ctr.Name,
			MachineID: machine.ID,
			Machine:   machine,
			ImageID:   &image.ID,
			Image:     image,
			VolumeID:  volume.ID,
			Volume:    volume,
			CreatedAt: time.Now(),
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
	machineNetwork, err := c.networkProvider.AllocateInterface(ctx)
	if err != nil {
		return fmt.Errorf("failed to allocate machine network: %w", err)
	}

	err = c.updateMachine(func(machine *types.Machine) error {
		machine.NetworkInterface = machineNetwork
		machine.NetworkInterface.MachineID = &machine.ID
		return c.db.WithContext(ctx).Select("MachineID").Save(machine.NetworkInterface).Error
	})
	if err != nil {
		return fmt.Errorf("failed to claim network interface for a machine: %w", err)
	}

	return nil
}
