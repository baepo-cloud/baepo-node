package machinecontroller

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"log/slog"
	"strings"
)

func (c *Controller) prepareMachine(ctx context.Context) error {
	if c.machine.Volume == nil {
		if err := c.prepareMachineVolume(ctx); err != nil {
			return fmt.Errorf("failed to prepare machine volume: %w", err)
		}
	}

	if c.machine.NetworkInterface == nil {
		if err := c.prepareMachineNetwork(ctx); err != nil {
			return fmt.Errorf("failed to prepare machine network: %w", err)
		}
	}

	return nil
}

func (c *Controller) prepareMachineVolume(ctx context.Context) error {
	c.log.Info("fetching image", slog.String("image-ref", c.machine.Spec.Image))
	imageRef, err := name.ParseReference(c.machine.Spec.Image)
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	image, err := remote.Image(imageRef)
	if err != nil {
		return fmt.Errorf("failed to fetch remote image: %w", err)
	}

	imageConfigFile, err := image.ConfigFile()
	if err != nil {
		return fmt.Errorf("failed to fetch image config file: %w", err)
	}

	c.machine.Spec.User = imageConfigFile.Config.User
	c.machine.Spec.WorkingDir = imageConfigFile.Config.WorkingDir
	for _, env := range imageConfigFile.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 1 {
			parts = append(parts, "")
		}
		c.machine.Spec.Env[parts[0]] = parts[1]
	}
	for key, value := range c.machine.Spec.Env {
		c.machine.Spec.Env[key] = value
	}
	c.machine.Spec.Command = append(imageConfigFile.Config.Entrypoint, imageConfigFile.Config.Cmd...)

	c.log.Info("creating machine volume")
	c.machine.Volume, err = c.volumeProvider.CreateVolume(ctx, image)
	if err != nil {
		return fmt.Errorf("failed to create machine volume: %w", err)
	}

	c.machine.Volume.MachineID = c.machine.ID
	err = c.db.WithContext(ctx).
		Model(&types.Volume{}).
		Where("id = ?", c.machine.Volume.ID).
		Update("machine_id", c.machine.ID).
		Error
	if err != nil {
		return fmt.Errorf("failed to claim volume for a machine: %w", err)
	}

	return nil
}

func (c *Controller) prepareMachineNetwork(ctx context.Context) error {
	machineNetwork, err := c.networkProvider.AllocateInterface(ctx)
	if err != nil {
		return fmt.Errorf("failed to allocate machine network: %w", err)
	}

	machineNetwork.MachineID = c.machine.ID
	c.machine.NetworkInterface = machineNetwork
	err = c.db.WithContext(ctx).
		Model(&types.NetworkInterface{}).
		Where("id = ?", machineNetwork.ID).
		Update("machine_id", c.machine.ID).
		Error
	if err != nil {
		return fmt.Errorf("failed to claim network interface for a machine: %w", err)
	}

	return nil
}
