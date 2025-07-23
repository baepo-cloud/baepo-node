package imageprovider

import (
	"context"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/nrednav/cuid2"
	"gorm.io/gorm"
	"strings"
)

func (p *Provider) FetchDetails(ctx context.Context, opts types.ImageFetchOptions) (*types.Image, error) {
	ref, err := name.ParseReference(opts.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference: %w", err)
	}

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

	size, err := remoteImage.Size()
	if err != nil {
		return nil, fmt.Errorf("failed to get image size: %w", err)
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
		Volume: &types.Volume{
			ID:   cuid2.Generate(),
			Size: uint64((size / 1024 / 1024) + 1024), // img size in mb + 1GiB,
		},
	}
	for _, env := range configFile.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 1 {
			parts = append(parts, "")
		}
		image.Spec.Env[parts[0]] = parts[1]
	}

	return image, nil
}
