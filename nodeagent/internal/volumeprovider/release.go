package volumeprovider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
)

func (p *Provider) Release(ctx context.Context, volume *types.Volume) error {
	if volume.ReleasedAt != nil {
		return nil
	}

	err := p.runCmd(ctx, "lvremove", "-y", fmt.Sprintf("%v/%v", p.volumeGroup, volume.ID))
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "failed to find logical volume") {
		return err
	}

	volume.ReleasedAt = typeutil.Ptr(time.Now())
	if err = p.db.WithContext(ctx).Select("ReleasedAt").Save(&volume).Error; err != nil {
		return fmt.Errorf("failed to persist volume changes: %w", err)
	}

	return nil
}
