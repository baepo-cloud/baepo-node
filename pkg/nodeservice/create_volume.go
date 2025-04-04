package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/pkg/types"
	"github.com/nrednav/cuid2"
	"os/exec"
)

func (s *Service) CreateVolume(ctx context.Context) (*types.NodeVolume, error) {
	volume := &types.NodeVolume{
		ID:       cuid2.Generate(),
		ReadOnly: false,
	}
	volume.Path = fmt.Sprintf("/dev/%s/%s", s.volumeGroup, volume.ID)
	baseVolumePath := fmt.Sprintf("/dev/%s/%s", s.volumeGroup, s.baseVolume)
	cmd := exec.CommandContext(ctx, "/usr/bin/lvcreate", "-s", "-n", volume.ID, baseVolumePath, "-L", "500M")
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return volume, nil
}
