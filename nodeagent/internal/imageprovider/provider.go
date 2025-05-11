package imageprovider

import (
	"bytes"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"gorm.io/gorm"
	"log/slog"
	"os/exec"
)

type Provider struct {
	logger         *slog.Logger
	db             *gorm.DB
	volumeProvider types.VolumeProvider
}

var _ types.ImageProvider = (*Provider)(nil)

func New(db *gorm.DB, volumeProvider types.VolumeProvider) *Provider {
	return &Provider{
		logger:         slog.With(slog.String("component", "imageprovider")),
		db:             db,
		volumeProvider: volumeProvider,
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
