package types

import (
	"context"
	"time"
)

type (
	ContainerStateChangedEvent struct {
		ContainerName    string
		Healthy          bool
		HealthcheckError error
		RestartCount     int32
		StartedAt        *time.Time
		ExitError        error
		ExitCode         *int32
		ExitedAt         *time.Time
		Timestamp        time.Time
	}

	ContainerService interface {
		Events(ctx context.Context) <-chan any
	}
)
