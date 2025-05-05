package imageprovider

import (
	"context"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/nrednav/cuid2"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/sys/unix"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (p *Provider) Fetch(ctx context.Context, opts types.ImageFetchOptions) (*types.Image, error) {
	ref, err := name.ParseReference(opts.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference: %w", err)
	}

	p.logger.Info("fetching image", slog.String("image-ref", ref.String()))
	remoteImage, err := remote.Image(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote image: %w", err)
	}

	digest, err := remoteImage.Digest()
	if err != nil {
		return nil, fmt.Errorf("failed to get image digest: %w", err)
	}

	var image *types.Image
	err = p.db.WithContext(ctx).Joins("Volume").First(&image, "digest = ?", digest.String()).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to find image in db: %w", err)
	} else if err == nil {
		return image, nil
	}

	configFile, err := remoteImage.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image config file: %w", err)
	}

	image = &types.Image{
		ID:     cuid2.Generate(),
		Digest: digest.String(),
		Name:   ref.String(),
		Spec: &types.ImageSpec{
			User:       configFile.Config.User,
			WorkingDir: configFile.Config.WorkingDir,
			Env:        map[string]string{},
			Command:    append(configFile.Config.Entrypoint, configFile.Config.Cmd...),
		},
	}
	for _, env := range configFile.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 1 {
			parts = append(parts, "")
		}
		image.Spec.Env[parts[0]] = parts[1]
	}

	volume, err := p.createVolumeFromImage(ctx, remoteImage)
	if err != nil {
		return nil, err
	}

	image.VolumeID = volume.ID
	image.Volume = volume
	if err = p.db.WithContext(ctx).Create(&image).Error; err != nil {
		return nil, fmt.Errorf("failed to create image in db: %w", err)
	}

	return image, nil
}

func (p *Provider) createVolumeFromImage(ctx context.Context, image v1.Image) (*types.Volume, error) {
	tmpDir, err := os.MkdirTemp("", "image-*")
	if err != nil {
		return nil, fmt.Errorf("faield to create tmp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	size, err := image.Size()
	if err != nil {
		return nil, fmt.Errorf("failed to get image size: %w", err)
	}

	volume, err := p.volumeProvider.Create(ctx, types.VolumeCreateOptions{
		Size: uint64((size / 1024 / 1024) + 1024), // img size in mb + 1GiB,
	})

	outputDir := filepath.Join(tmpDir, "output")
	if err = os.Mkdir(outputDir, 0644); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
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
