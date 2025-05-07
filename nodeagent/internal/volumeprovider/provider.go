package volumeprovider

import (
	"bytes"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"gorm.io/gorm"
	"os/exec"
)

type Provider struct {
	db          *gorm.DB
	volumeGroup string
}

var _ types.VolumeProvider = (*Provider)(nil)

func New(db *gorm.DB, config *types.Config) *Provider {
	return &Provider{
		db:          db,
		volumeGroup: config.VolumeGroup,
	}
}

func (p *Provider) runCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}

	return nil
}
