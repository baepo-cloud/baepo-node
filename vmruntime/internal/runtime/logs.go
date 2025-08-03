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

type logsManager struct {
	manager         *logmanager.Manager
	runtime         *Runtime
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	containerLogSeq atomic.Uint64
}

func newLogsManager(runtime *Runtime) (*logsManager, error) {
	manager, err := logmanager.New(path.Join(runtime.config.WorkingDir, "logs.json"))
	if err != nil {
		return nil, err
	}

	logManager := &logsManager{manager: manager, runtime: runtime}
	logManager.ctx, logManager.cancel = context.WithCancel(context.Background())
	if err = logManager.startSerialSocketServer(); err != nil {
		return nil, fmt.Errorf("failed to start serial socket server: %v", err)
	}

	logManager.watchInitLogs()
	return logManager, nil
}

func (m *logsManager) Stop() {
	m.cancel()
	m.wg.Wait()
	_ = m.manager.Close()
}

func (m *logsManager) GetSerialSocketPath() string {
	return path.Join(m.runtime.config.WorkingDir, "serial.socket")
}

func (m *logsManager) startSerialSocketServer() error {
	lis, err := net.Listen("unix", m.GetSerialSocketPath())
	if err != nil {
		return err
	}

	go func() {
		<-m.ctx.Done()
		_ = lis.Close()
	}()

	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				fmt.Println(err)
				break
			}

			go m.handleSerialConn(conn)
		}
	}()

	return nil
}

func (m *logsManager) watchInitLogs() {
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
				_ = m.connectToInitLogStream()
			}
		}
	}()
}

func (m *logsManager) connectToInitLogStream() error {
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
}

func (m *logsManager) handleSerialConn(conn net.Conn) error {
	scanner := bufio.NewScanner(conn)
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

	return scanner.Err()
}
