package nodeservice

import (
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/pkg/types"
	"os/exec"
)

func (s *Service) DeleteVolume(ctx context.Context, volume *types.NodeVolume) error {
	cmd := exec.CommandContext(ctx, "/usr/bin/lvremove", "-y", volume.Path)
	if err := cmd.Run(); err != nil {
		output, _ := cmd.Output()
		return fmt.Errorf("failed to remove logical volume (output: %v): %w", string(output), err)
	}
	return nil
}
