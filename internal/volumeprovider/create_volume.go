package volumeprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/nrednav/cuid2"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func (p *Provider) CreateVolume(ctx context.Context, image v1.Image) (*types.Volume, error) {
	tmpDir, err := os.MkdirTemp("", "image-*")
	if err != nil {
		return nil, fmt.Errorf("faield to create tmp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	imgSize, err := image.Size()
	if err != nil {
		return nil, fmt.Errorf("failed to get image size: %w", err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	if err = os.Mkdir(outputDir, 0644); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}

	volume := &types.Volume{
		ID:       cuid2.Generate(),
		ReadOnly: false,
		Size:     uint64((imgSize / 1024 / 1024) + 1024), // img size in mb + 1GiB,
	}
	volume.Path = fmt.Sprintf("/dev/%v/%v", p.volumeGroup, volume.ID)
	if err = p.db.WithContext(ctx).Create(&volume).Error; err != nil {
		return nil, fmt.Errorf("failed to create volume in database: %w", err)
	}

	err = p.runCmd(ctx, "/usr/bin/lvcreate",
		"-y", "--virtualsize", fmt.Sprintf("%vM", volume.Size), "--thin",
		"-n", volume.ID, fmt.Sprintf("%v/thinpool", p.volumeGroup),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create logical volume: %w", err)
	}

	err = p.runCmd(ctx, "mkfs.ext4", volume.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume fs: %w", err)
	}

	if err = unix.Mount(volume.Path, outputDir, "ext4", 0, ""); err != nil {
		return nil, fmt.Errorf("failed to mount volume: %w", err)
	}

	if err = p.extractImage(image, tmpDir, outputDir); err != nil {
		return nil, fmt.Errorf("failed to extract image: %w", err)
	}

	if err = unix.Unmount(outputDir, 0); err != nil {
		return nil, fmt.Errorf("failed to unmount volume: %w", err)
	}

	return volume, nil
}

func (p *Provider) extractImage(image v1.Image, tmpDir, outputDir string) error {
	layers, err := image.Layers()
	if err != nil {
		return fmt.Errorf("failed to get layers: %v", err)
	}

	layersDir := filepath.Join(tmpDir, "layers")
	if err = os.Mkdir(layersDir, 0644); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}

	wg := pool.New().WithErrors().WithMaxGoroutines(4)
	for i, layer := range layers {
		wg.Go(func() error {
			rc, err := layer.Compressed()
			if err != nil {
				return fmt.Errorf("failed to get layer %d: %v", i, err)
			}
			defer rc.Close()

			layerPath := filepath.Join(layersDir, fmt.Sprintf("layer_%d.tar", i))
			layerFile, err := os.Create(layerPath)
			if err != nil {
				return fmt.Errorf("failed to create layer file %d: %v", i, err)
			}

			if _, err = io.Copy(layerFile, rc); err != nil {
				layerFile.Close()
				return fmt.Errorf("failed to write layer %d: %v", i, err)
			}
			layerFile.Close()
			return nil
		})
	}

	if err = wg.Wait(); err != nil {
		return fmt.Errorf("failed to download image layers: %v", err)
	}

	for i := range layers {
		layerPath := filepath.Join(layersDir, fmt.Sprintf("layer_%d.tar", i))
		extractCmd := exec.Command("tar", "-xf", layerPath, "-C", outputDir)
		if err = extractCmd.Run(); err != nil {
			return fmt.Errorf("failed to extract layer %d: %v", i, err)
		}
	}

	return nil
}
