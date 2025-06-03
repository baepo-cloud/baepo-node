package types

import (
	"context"
	"time"
)

type (
	Volume struct {
		ID        string `gorm:"primaryKey"`
		Path      string
		Size      uint64
		SourceID  *string
		Source    *Volume
		CreatedAt time.Time
		DeletedAt *time.Time
	}

	VolumeCreateOptions struct {
		Size   uint64
		Source *Volume
	}

	VolumeProvider interface {
		Create(ctx context.Context, opts VolumeCreateOptions) (*Volume, error)

		Delete(ctx context.Context, volume *Volume) error
	}
)
