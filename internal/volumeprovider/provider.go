package volumeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/nrednav/cuid2"
	"os/exec"
)

type Provider struct {
	volumeGroup string
	baseVolume  string
}

var _ types.VolumeProvider = (*Provider)(nil)

func New(volumeGroup, baseVolume string) *Provider {
	return &Provider{
		volumeGroup: volumeGroup,
		baseVolume:  baseVolume,
	}
}

func (s *Provider) CreateVolume(ctx context.Context) (*types.Volume, error) {
	volume := &types.Volume{
		ID:       cuid2.Generate(),
		ReadOnly: false,
	}
	volume.Path = fmt.Sprintf("/dev/%s/%s", s.volumeGroup, volume.ID)
	baseVolumePath := fmt.Sprintf("/dev/%s/%s", s.volumeGroup, s.baseVolume)
	cmd := exec.CommandContext(ctx, "/usr/bin/lvcreate", "-s", "-n", volume.ID, baseVolumePath, "-L", "500M")
	if err := cmd.Run(); err != nil {
		output, outputErr := cmd.Output()
		if outputErr == nil {
			return nil, fmt.Errorf("%w: %s", err, string(output))
		}

		return nil, err
	}

	return volume, nil
}

func (s *Provider) DeleteVolume(ctx context.Context, volume *types.Volume) error {
	cmd := exec.CommandContext(ctx, "/usr/bin/lvremove", "-y", volume.Path)
	if err := cmd.Run(); err != nil {
		output, outputErr := cmd.Output()
		if outputErr == nil {
			return fmt.Errorf("%w: %s", err, string(output))
		}

		return err
	}

	return nil
}
