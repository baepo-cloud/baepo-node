package types

import (
	"context"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type (
	Volume struct {
		ID       string
		Path     string
		ReadOnly bool
	}

	VolumeProvider interface {
		CreateVolume(ctx context.Context, image v1.Image) (*Volume, error)

		DeleteVolume(ctx context.Context, volume *Volume) error
	}
)
