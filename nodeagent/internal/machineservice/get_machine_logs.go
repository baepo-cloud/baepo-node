package machineservice

import (
	"bufio"
	"context"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"os"
	"path"
)

func (s *Service) GetMachineLogs(ctx context.Context, opts types.MachineGetMachineLogsOptions) (<-chan types.MachineLog, error) {
	file, err := os.Open(path.Join(s.config.StorageDirectory, "machines", opts.MachineID, "machine.log"))
	if err != nil {
		return nil, err
	}

	logs := make(chan types.MachineLog)
	go func() {
		sc := bufio.NewScanner(file)
		defer func() {
			file.Close()
			if !opts.Follow || sc.Err() != nil {
				close(logs)
			}
		}()

		for sc.Scan() {
			if ctx.Err() != nil {
				break
			}

			logs <- types.MachineLog{
				Content: sc.Bytes(),
			}
		}
	}()

	return logs, nil
}
