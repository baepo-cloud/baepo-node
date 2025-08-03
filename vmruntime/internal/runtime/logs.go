package runtime

import (
	"bufio"
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/logmanager"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"io"
	"os"
	"path"
	"sync"
	"time"
)

type logsManager struct {
	manager *logmanager.Manager
	runtime *Runtime
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func newLogsManager(runtime *Runtime) (*logsManager, error) {
	manager, err := logmanager.New(path.Join(runtime.config.WorkingDir, "logs.json"))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &logsManager{
		manager: manager,
		runtime: runtime,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

func (m *logsManager) Stop() {
	m.cancel()
	m.wg.Wait()
	_ = m.manager.Close()
}

func (m *logsManager) WatchInitLogs() {
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
				err := m.connectToInitLogStream()
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "failed to connect to init log stream: %v\n", err)
					ticker.Reset(time.Second)
				}
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
	for stream.Receive() {
		log := stream.Msg()
		_ = m.manager.WriteLog(logmanager.Entry{
			Source:      logmanager.ContainerLogEntrySource,
			ContainerID: &log.ContainerId,
			Timestamp:   log.Timestamp.AsTime(),
			Message:     string(log.Content),
			Stderr:      log.Error,
		})
	}

	return stream.Err()
}

func (m *logsManager) HandleHypervisorOutput(reader io.Reader, stderr bool) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		_ = m.manager.WriteLog(logmanager.Entry{
			Source:  logmanager.MachineLogEntrySource,
			Message: line,
			Stderr:  stderr,
		})
	}
	
	return scanner.Err()
}
