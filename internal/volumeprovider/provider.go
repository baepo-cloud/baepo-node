package volumeprovider

import (
	"bytes"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"os/exec"
)

type Provider struct {
	volumeGroup string
}

var _ types.VolumeProvider = (*Provider)(nil)

func New(volumeGroup string) *Provider {
	return &Provider{
		volumeGroup: volumeGroup,
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
