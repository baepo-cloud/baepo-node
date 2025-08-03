package logmanager

import (
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

func New(logPath string) (*Manager, error) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &Manager{
		logFile: file,
		encoder: json.NewEncoder(file),
		logPath: logPath,
	}, nil
}

func (lm *Manager) WriteLog(entry Entry) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	entry.Timestamp = time.Now()
	return lm.encoder.Encode(entry)
}

func (lm *Manager) Close() error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	return lm.logFile.Close()
}
