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

func (s *Service) GetContainerLogs(ctx context.Context, opts types.MachineGetContainerLogsOptions) (<-chan types.MachineContainerLog, error) {
	if _, ok := s.machineControllers.Get(opts.MachineID); !ok {
		return s.readContainerLogsFromFile(ctx, opts)
	}

	runtimeClient, closeRuntimeClient := s.runtimeService.GetClient(opts.MachineID)
	stream, err := runtimeClient.GetContainerLogs(ctx, connect.NewRequest(&nodev1pb.RuntimeGetContainerLogsRequest{
		ContainerId: opts.ContainerID,
		Follow:      opts.Follow,
	}))
	if err != nil {
		closeRuntimeClient()
		return nil, err
	}

	logs := make(chan types.MachineContainerLog)
	go func() {
		defer closeRuntimeClient()

		for stream.Receive() {
			log := stream.Msg()
			logs <- types.MachineContainerLog{
				ContainerID: log.ContainerId,
				Error:       log.Error,
				Content:     log.Content,
				Timestamp:   log.Timestamp.AsTime(),
			}
		}
	}()
	return logs, nil
}

func (s *Service) readContainerLogsFromFile(ctx context.Context, opts types.MachineGetContainerLogsOptions) (<-chan types.MachineContainerLog, error) {
	logManager := logmanager.New(path.Join(s.runtimeService.GetMachineDirectory(opts.MachineID), "logs.json"))
	entries, err := logManager.ReadLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	logs := make(chan types.MachineContainerLog)
	go func() {
		defer close(logs)

		for {
			select {
			case <-ctx.Done():
				return
			case log, ok := <-entries:
				if !ok {
					return
				} else if log.Source != logmanager.ContainerLogEntrySource {
					continue
				}

				logs <- types.MachineContainerLog{
					ContainerID: *log.ContainerID,
					Error:       log.Stderr,
					Content:     []byte(log.Message),
					Timestamp:   log.Timestamp,
				}
			}
		}
	}()
	return logs, nil
}
