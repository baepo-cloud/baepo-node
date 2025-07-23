package types

import (
	"context"
	"errors"
	"time"
)

type (
	Volume struct {
		ID          string `gorm:"primaryKey"`
		Size        uint64
		Path        *string
		SourceID    *string
		Source      *Volume
		AllocatedAt *time.Time
		ReleasedAt  *time.Time
		CreatedAt   time.Time
	}

	VolumeProvider interface {
		Allocate(ctx context.Context, volume *Volume) error

		Release(ctx context.Context, volume *Volume) error
	}
)

var ErrVolumeAlreadyAllocated = errors.New("volume already allocated")
