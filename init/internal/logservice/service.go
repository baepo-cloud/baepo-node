package logservice

import (
	"encoding/json"
	"fmt"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"github.com/nrednav/cuid2"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"sync"
)

type Service struct {
	storageDirectory string
	writer           *json.Encoder
	writerMutex      sync.Mutex
	handlersMutex    sync.RWMutex
	handlers         map[string]func(entry *types.LogEntry)
}

var _ types.LogService = (*Service)(nil)

func New(storageDirectory string) (*Service, error) {
	if err := os.MkdirAll(storageDirectory, 0644); err != nil {
		return nil, fmt.Errorf("failed to create log manager storage directory: %w", err)
	}

	file, err := os.Create(filepath.Join(storageDirectory, "logs.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return &Service{
		storageDirectory: storageDirectory,
		writer:           json.NewEncoder(file),
		handlers:         make(map[string]func(entry *types.LogEntry)),
	}, nil
}

func (s *Service) Write(entry *types.LogEntry) error {
	s.writerMutex.Lock()
	defer s.writerMutex.Unlock()

	if err := s.writer.Encode(&entry); err != nil {
		return err
	}

	_, _ = unix.Write(int(entry.Fd), []byte(entry.Content))

	s.handlersMutex.RLock()
	defer s.handlersMutex.RUnlock()
	for _, handler := range s.handlers {
		handler(entry)
	}

	return nil
}

func (s *Service) AddHandler(handler func(entry *types.LogEntry)) func() {
	s.handlersMutex.Lock()
	defer s.handlersMutex.Unlock()

	handlerID := cuid2.Generate()
	s.handlers[handlerID] = handler

	return func() {
		s.handlersMutex.Lock()
		defer s.handlersMutex.Unlock()

		delete(s.handlers, handlerID)
	}
}
