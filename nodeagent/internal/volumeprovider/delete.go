package volumeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"gorm.io/gorm"
	"strings"
	"time"
)

func (p *Provider) Delete(ctx context.Context, volume *types.Volume) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		volume.DeletedAt = typeutil.Ptr(time.Now())
		err := p.db.WithContext(ctx).Save(&volume).Error
		if err != nil {
			return fmt.Errorf("failed to save volume in database: %w", err)
		}

		err = p.runCmd(ctx, "/usr/bin/lvremove", "-y", fmt.Sprintf("%v/%v", p.volumeGroup, volume.ID))
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "failed to find logical volume") {
			return err
		}

		return nil
	})
}
