package machineservice

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/logmanager"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	nodev1pb "github.com/baepo-cloud/baepo-proto/go/baepo/node/v1"
	"path"
)

func (s *Service) GetMachineLogs(ctx context.Context, opts types.MachineGetMachineLogsOptions) (<-chan types.MachineLog, error) {
	if _, ok := s.machineControllers.Get(opts.MachineID); !ok {
		return s.readMachineLogsFromFile(ctx, opts.MachineID)
	}

	runtimeClient, closeRuntimeClient := s.runtimeService.GetClient(opts.MachineID)
	stream, err := runtimeClient.GetLogs(ctx, connect.NewRequest(&nodev1pb.RuntimeGetLogsRequest{
		Follow: opts.Follow,
	}))
	if err != nil {
		closeRuntimeClient()
		return nil, err
	}

	logs := make(chan types.MachineLog)
	go func() {
		defer closeRuntimeClient()

		for stream.Receive() {
			log := stream.Msg()
			logs <- types.MachineLog{
				Content: log.Content,
			}
		}
	}()
	return logs, nil
}

func (s *Service) readMachineLogsFromFile(ctx context.Context, machineID string) (<-chan types.MachineLog, error) {
	logManager := logmanager.New(path.Join(s.runtimeService.GetMachineDirectory(machineID), "logs.json"))
	entries, err := logManager.ReadLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	logs := make(chan types.MachineLog)
	go func() {
		defer close(logs)

		for {
			select {
			case <-ctx.Done():
				return
			case logEntry, ok := <-entries:
				if !ok {
					return
				} else if logEntry.Source != logmanager.MachineLogEntrySource {
					continue
				}

				logs <- types.MachineLog{
					Content: []byte(logEntry.Message),
				}
			}
		}
	}()
	return logs, nil
}
