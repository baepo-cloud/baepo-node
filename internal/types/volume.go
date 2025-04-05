package types

import "context"

type (
	Volume struct {
		ID       string
		Path     string
		ReadOnly bool
	}

	VolumeProvider interface {
		CreateVolume(ctx context.Context) (*Volume, error)

		DeleteVolume(ctx context.Context, volume *Volume) error
	}
)
