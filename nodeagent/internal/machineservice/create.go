package machineservice

import (
	"context"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/nrednav/cuid2"
	"gorm.io/gorm"
	"log/slog"
)

func (s *Service) Create(ctx context.Context, opts types.MachineCreateOptions) (*types.Machine, error) {
	s.log.Info("requesting machine creation", slog.String("machine-id", opts.MachineID))
	machine := &types.Machine{
		ID:           opts.MachineID,
		State:        coretypes.MachineStatePending,
		DesiredState: opts.DesiredState,
		Spec:         opts.Spec,
		Containers:   make([]*types.Container, len(opts.Containers)),
	}

	for index, containerOpt := range opts.Containers {
		image, err := s.imageProvider.FetchDetails(ctx, types.ImageFetchOptions{
			Image: containerOpt.Spec.Image,
		})
		if err != nil {
			return nil, fmt.Errorf("failed ot fetch container image details (%v): %v", containerOpt.Spec.Image, err)
		}

		container := &types.Container{
			ID:        containerOpt.ContainerID,
			MachineID: machine.ID,
			Spec:      (*types.ContainerSpec)(containerOpt.Spec),
		}
		volume := &types.Volume{
			ID:       cuid2.Generate(),
			Size:     1024, // 1 gib
			SourceID: &image.Volume.ID,
			Source:   image.Volume,
		}
		machine.Containers[index] = container
		machine.Volumes = append(machine.Volumes, &types.MachineVolume{
			ID:          cuid2.Generate(),
			Position:    len(machine.Volumes),
			MachineID:   machine.ID,
			ContainerID: container.ID,
			ImageID:     &image.ID,
			Image:       image,
			VolumeID:    volume.ID,
			Volume:      volume,
		})
	}

	err := s.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Create(&machine).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %w", err)
	}

	s.machineControllers.Set(machine.ID, s.newMachineController(machine))
	return machine, nil
}
