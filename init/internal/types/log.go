package types

import (
	"context"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"io"
	"time"
)

type (
	LogEntry struct {
		Timestamp     time.Time
		ContainerName string
		Fd            uint32
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
