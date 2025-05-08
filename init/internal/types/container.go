package types

import (
	"context"
	"time"
)

type (
	ContainerEventType string

	ContainerEvent interface {
		Type() ContainerEventType
	}

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
		Events(ctx context.Context) <-chan ContainerEvent
	}
)

const (
	ContainerStateChangedEventType ContainerEventType = "container_state_changed"
)

func (ContainerStateChangedEvent) Type() ContainerEventType {
	return ContainerStateChangedEventType
}
