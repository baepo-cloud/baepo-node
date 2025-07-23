package volumeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"time"
)

func (p *Provider) Allocate(ctx context.Context, volume *types.Volume) error {
	if volume.AllocatedAt != nil {
		return types.ErrVolumeAlreadyAllocated
	}

	volume.Path = typeutil.Ptr(fmt.Sprintf("/dev/%v/%v", p.volumeGroup, volume.ID))

	var args []string
	if volume.Source != nil {
		if volume.Source.Path == nil {
			return fmt.Errorf(
				"cannot allocate a volume from source %v with path a null path (source volume is probably not allocated)",
				volume.Source.ID,
			)
		}

		// Create a snapshot of the source volume
		args = []string{
			"-y",
			"--size", fmt.Sprintf("%vM", volume.Size),
			"--snapshot",
			"--name", volume.ID,
			*volume.Source.Path,
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

	volume.AllocatedAt = typeutil.Ptr(time.Now())
	if err = p.db.WithContext(ctx).Select("Path", "AllocatedAt").Save(&volume).Error; err != nil {
		return fmt.Errorf("failed to persist volume changes: %w", err)
	}

	return nil
}
