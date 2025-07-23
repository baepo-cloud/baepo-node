package imageprovider

import (
	"context"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/sys/unix"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func (p *Provider) Pull(ctx context.Context, image *types.Image) error {
	if image.PulledAt != nil {
		return nil
	}

	log := p.logger.With(slog.String("image-id", image.ID), slog.String("image-name", image.Name))
	log.Info("pulling image")

	tmpDir, err := os.MkdirTemp("", "image-*")
	if err != nil {
		return fmt.Errorf("failed to create tmp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	imageRef, err := name.ParseReference(image.Name)
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	remoteImage, err := remote.Image(imageRef)
	if err != nil {
		return fmt.Errorf("failed to fetch remote image: %w", err)
	}

	if err = p.volumeProvider.Allocate(ctx, image.Volume); err != nil && !errors.Is(err, types.ErrVolumeAlreadyAllocated) {
		return fmt.Errorf("failed to allocate volume %v: %v", image.VolumeID, err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	if err = os.Mkdir(outputDir, 0644); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}

	err = p.runCmd(ctx, "mkfs.ext4", *image.Volume.Path)
	if err != nil {
		return fmt.Errorf("failed to create volume fs: %w", err)
	}

	if err = unix.Mount(*image.Volume.Path, outputDir, "ext4", 0, ""); err != nil {
		return fmt.Errorf("failed to mount volume: %w", err)
	}

	if err = p.extractImage(remoteImage, tmpDir, outputDir); err != nil {
		return fmt.Errorf("failed to extract image: %w", err)
	}

	if err = unix.Unmount(outputDir, 0); err != nil {
		return fmt.Errorf("failed to unmount volume: %w", err)
	}

	image.PulledAt = typeutil.Ptr(time.Now())
	if err = p.db.WithContext(ctx).Select("PulledAt").Save(&image).Error; err != nil {
		return fmt.Errorf("failed to persist image changes: %w", err)
	}

	return nil
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
