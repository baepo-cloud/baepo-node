package logservice

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"io"
	"log/slog"
	"os"
	"path"
)

func (s *Service) Read(ctx context.Context) (<-chan *types.LogEntry, error) {
	logChan := make(chan *types.LogEntry)

	go func() {
		defer close(logChan)

		if err := s.readHistoricalLogs(ctx, logChan); err != nil {
			slog.Error("failed to read historical logs", slog.Any("error", err))
			return
		}

		unsubscribe := s.AddHandler(func(entry *types.LogEntry) {
			logChan <- entry
		})
		defer unsubscribe()

		<-ctx.Done()
	}()

	return logChan, nil
}

func (s *Service) readHistoricalLogs(ctx context.Context, logChan chan<- *types.LogEntry) error {
	file, err := os.Open(path.Join(s.storageDirectory, "logs.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var entry types.LogEntry
			err = decoder.Decode(&entry)
			if err == io.EOF {
				return nil
			} else if err != nil {
				return fmt.Errorf("failed to decode log entry: %w", err)
			}

			logChan <- &entry
		}
	}
}
