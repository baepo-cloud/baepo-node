package logmanager

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type (
	EntrySource string

	Entry struct {
		Source      EntrySource `json:"source"`
		ContainerID *string     `json:"container_id,omitempty"`
		Timestamp   time.Time   `json:"timestamp"`
		Message     string      `json:"message"`
		Stderr      bool        `json:"stderr"`
	}

	Manager struct {
		logFile *os.File
		encoder *json.Encoder
		mutex   sync.Mutex
		logPath string
	}
)

const (
	MachineLogEntrySource   EntrySource = "machine"
	ContainerLogEntrySource EntrySource = "container"
)

func New(logPath string) *Manager {
	return &Manager{logPath: logPath}
}

func (m *Manager) WriteLog(entry Entry) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.logFile == nil {
		file, err := os.OpenFile(m.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		m.logFile = file
	}
	if m.encoder == nil {
		m.encoder = json.NewEncoder(m.logFile)
	}

	entry.Timestamp = time.Now()
	return m.encoder.Encode(entry)
}

func (m *Manager) ReadLogs(ctx context.Context) (<-chan Entry, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	file, err := os.Open(m.logPath)
	if err != nil {
		return nil, err
	}

	entries := make(chan Entry)
	go func() {
		defer file.Close()
		defer close(entries)
		decoder := json.NewDecoder(file)

		for {
			if ctx.Err() != nil {
				return
			}

			var entry Entry
			if err = decoder.Decode(&entry); err != nil {
				return
			}

			entries <- entry
		}
	}()

	return entries, nil
}

func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.logFile != nil {
		if err := m.logFile.Close(); err != nil {
			return err
		}
	}

	return nil
}
