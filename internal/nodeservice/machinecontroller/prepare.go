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
	machine := c.GetMachine()
	if machine.Volume == nil {
		if err := c.prepareMachineVolume(ctx); err != nil {
			return fmt.Errorf("failed to prepare machine volume: %w", err)
		}
	}

	if machine.NetworkInterface == nil {
		if err := c.prepareMachineNetwork(ctx); err != nil {
			return fmt.Errorf("failed to prepare machine network: %w", err)
		}
	}

	return nil
}

func (c *Controller) prepareMachineVolume(ctx context.Context) error {
	machine := c.GetMachine()

	c.log.Info("fetching image", slog.String("image-ref", machine.Spec.Image))
	imageRef, err := name.ParseReference(machine.Spec.Image)
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

	c.log.Info("creating machine volume")
	volume, err := c.volumeProvider.CreateVolume(ctx, image)
	if err != nil {
		return fmt.Errorf("failed to create machine volume: %w", err)
	}

	err = c.updateMachine(func(machine *types.Machine) error {
		machine.Spec.User = imageConfigFile.Config.User
		machine.Spec.WorkingDir = imageConfigFile.Config.WorkingDir
		for _, env := range imageConfigFile.Config.Env {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 1 {
				parts = append(parts, "")
			}
			machine.Spec.Env[parts[0]] = parts[1]
		}
		for key, value := range machine.Spec.Env {
			machine.Spec.Env[key] = value
		}
		machine.Spec.Command = append(imageConfigFile.Config.Entrypoint, imageConfigFile.Config.Cmd...)
		machine.Volume = volume
		machine.Volume.MachineID = &machine.ID
		return c.db.WithContext(ctx).Select("MachineID").Save(machine.Volume).Error
	})
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
