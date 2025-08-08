package runtime

import (
	"bufio"
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/logmanager"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"github.com/nrednav/cuid2"
	"maps"
	"net"
	"path"
	"sync"
	"sync/atomic"
	"time"
)

type logManager struct {
	manager         *logmanager.Manager
	runtime         *Runtime
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	containerLogSeq atomic.Uint64
	logHandlers     map[string]func(entry logmanager.Entry)
	logHandlersLock sync.RWMutex
}

func newLogManager(runtime *Runtime) *logManager {
	manager := &logManager{
		manager:     logmanager.New(path.Join(runtime.config.WorkingDir, "logs.json")),
		runtime:     runtime,
		logHandlers: make(map[string]func(entry logmanager.Entry)),
	}
	manager.ctx, manager.cancel = context.WithCancel(context.Background())
	return manager
}

func (m *logManager) Stop() {
	m.cancel()
	m.wg.Wait()
	_ = m.manager.Close()
}

func (m *logManager) GetSerialSocketPath() string {
	return path.Join(m.runtime.config.WorkingDir, "serial.socket")
}

func (m *logManager) ListenSerialSocket() {
	m.startListener(func() error {
		c, err := net.Dial("unix", m.GetSerialSocketPath())
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(c)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			_ = m.writeLog(logmanager.Entry{
				Source:  logmanager.MachineLogEntrySource,
				Message: line,
			})
		}
		return c.Close()
	})
}

func (m *logManager) ListenInitLogs() {
	m.startListener(func() error {
		client, closeClient := m.runtime.newInitClient()
		defer closeClient()

		stream, err := client.GetLogs(m.ctx, connect.NewRequest(&nodev1pb.InitGetLogsRequest{}))
		if err != nil {
			return fmt.Errorf("failed to get init logs: %w", err)
		}

		defer stream.Close()
		seq := uint64(0)
		for stream.Receive() {
			seq += 1
			if seq <= m.containerLogSeq.Load() {
				continue
			}

			log := stream.Msg()
			_ = m.writeLog(logmanager.Entry{
				Source:      logmanager.ContainerLogEntrySource,
				ContainerID: &log.ContainerId,
				Timestamp:   log.Timestamp.AsTime(),
				Message:     string(log.Content),
				Stderr:      log.Error,
			})
			m.containerLogSeq.Add(1)
		}

		return stream.Err()
	})
}

func (m *logManager) HandleLogs(handler func(entry logmanager.Entry)) func() {
	m.logHandlersLock.Lock()
	defer m.logHandlersLock.Unlock()

	handlerID := cuid2.Generate()
	m.logHandlers[handlerID] = handler
	return func() {
		m.logHandlersLock.Lock()
		defer m.logHandlersLock.Unlock()

		delete(m.logHandlers, handlerID)
	}
}

func (m *logManager) startListener(listen func() error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	m.wg.Add(1)

	go func() {
		defer func() {
			ticker.Stop()
			m.wg.Done()
		}()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				_ = listen()
			}
		}
	}()
}

func (m *logManager) ReadLogs(ctx context.Context) (<-chan logmanager.Entry, error) {
	return m.manager.ReadLogs(ctx)
}

func (m *logManager) writeLog(entry logmanager.Entry) error {
	if err := m.manager.WriteLog(entry); err != nil {
		return err
	}

	m.logHandlersLock.RLock()
	handlers := maps.Values(m.logHandlers)
	m.logHandlersLock.RUnlock()

	for handler := range handlers {
		handler(entry)
	}
	return nil
}
