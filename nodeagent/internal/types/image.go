package types

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"time"
)

type (
	Image struct {
		ID        string
		Digest    string `gorm:"unique"`
		Name      string
		Spec      *ImageSpec
		VolumeID  string
		Volume    *Volume
		PulledAt  *time.Time
		CreatedAt time.Time
	}

	ImageSpec struct {
		User       string
		WorkingDir string
		Env        map[string]string
		Command    []string
	}

	ImageFetchOptions struct {
		Image string
	}

	ImageProvider interface {
		FetchDetails(ctx context.Context, opts ImageFetchOptions) (*Image, error)

		Pull(ctx context.Context, image *Image) error
	}
)

func (*ImageSpec) GormDataType() string {
	return "jsonb"
}

func (s *ImageSpec) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &s)
}

func (s *ImageSpec) Value() (driver.Value, error) {
	return json.Marshal(s)
}
