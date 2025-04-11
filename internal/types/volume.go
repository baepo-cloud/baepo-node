package types

import (
	"context"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"time"
)

type (
	Volume struct {
		ID        string `gorm:"primaryKey"`
		MachineID string
		Path      string
		ReadOnly  bool
		Size      uint64
		CreatedAt time.Time
		DeletedAt *time.Time
	}

	VolumeProvider interface {
		CreateVolume(ctx context.Context, image v1.Image) (*Volume, error)

		DeleteVolume(ctx context.Context, volume *Volume) error
	}
)
