package runtime

import (
	"bufio"
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/logmanager"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
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
}

func newLogManager(runtime *Runtime) (*logManager, error) {
	coreManager, err := logmanager.New(path.Join(runtime.config.WorkingDir, "logs.json"))
	if err != nil {
		return nil, err
	}

	manager := &logManager{manager: coreManager, runtime: runtime}
	manager.ctx, manager.cancel = context.WithCancel(context.Background())
	return manager, nil
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
			fmt.Println(line)
			if line == "" {
				continue
			}

			_ = m.manager.WriteLog(logmanager.Entry{
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
			_ = m.manager.WriteLog(logmanager.Entry{
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
