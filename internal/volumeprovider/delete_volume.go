package volumeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
	"time"
)

func (p *Provider) DeleteVolume(ctx context.Context, volume *types.Volume) error {
	err := p.runCmd(ctx, "/usr/bin/lvremove", "-y", fmt.Sprintf("%v/%v", p.volumeGroup, volume.ID))
	if err != nil {
		return err
	}

	volume.DeletedAt = typeutil.Ptr(time.Now())
	err = p.db.WithContext(ctx).Save(&volume).Error
	if err != nil {
		return fmt.Errorf("failed to save volume in database: %w", err)
	}

	return err
}
