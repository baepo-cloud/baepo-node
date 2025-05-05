package volumeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/nrednav/cuid2"
	"gorm.io/gorm"
)

func (p *Provider) Create(ctx context.Context, opts types.VolumeCreateOptions) (*types.Volume, error) {
	volume := &types.Volume{
		ID:   cuid2.Generate(),
		Size: opts.Size,
	}
	volume.Path = fmt.Sprintf("/dev/%v/%v", p.volumeGroup, volume.ID)

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := p.db.WithContext(ctx).Create(&volume).Error; err != nil {
			return fmt.Errorf("failed to create volume in database: %w", err)
		}

		var args []string
		if opts.Source != nil {
			// Create a snapshot of the source volume
			args = []string{
				"-y",
				"--size", fmt.Sprintf("%vM", volume.Size),
				"--snapshot",
				"--name", volume.ID,
				opts.Source.Path,
			}
		} else {
			// Create a new thin provisioned volume
			args = []string{
				"-y",
				"--virtualsize", fmt.Sprintf("%vM", volume.Size),
				"--thin",
				"--name", volume.ID,
				fmt.Sprintf("%v/thinpool", p.volumeGroup),
			}
		}

		err := p.runCmd(ctx, "/usr/bin/lvcreate", args...)
		if err != nil {
			return fmt.Errorf("failed to create logical volume: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return volume, nil
}
