package volumeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
)

func (p *Provider) DeleteVolume(ctx context.Context, volume *types.Volume) error {
	return p.runCmd(ctx, "/usr/bin/lvremove", "-y", fmt.Sprintf("%v/%v", p.volumeGroup, volume.ID))
}
