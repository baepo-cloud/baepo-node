package types

import (
	"context"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"io"
	"time"
)

type (
	LogEntry struct {
		ContainerID   string
		ContainerName *string
		Timestamp     time.Time
		Error         bool
		Content       string
	}

	LogReadOptions struct {
		Follow bool
	}

	LogService interface {
		NewContainerLogWriter(config coretypes.InitContainerConfig) (io.Writer, io.Writer)

		AddHandler(handler func(entry *LogEntry)) func()

		Read(ctx context.Context, opts LogReadOptions) (<-chan *LogEntry, error)
	}
)
