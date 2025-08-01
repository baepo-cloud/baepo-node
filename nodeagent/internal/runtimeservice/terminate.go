package runtimeservice

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/emptypb"
	"os"
	"syscall"
	"time"
)

func (s *Service) Terminate(ctx context.Context, machineID string) error {
	client, closeClient := s.GetClient(machineID)
	defer closeClient()

	res, err := client.GetState(ctx, connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	process, err := os.FindProcess(int(res.Msg.Pid))
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", res.Msg.Pid, err)
	}

	waitChan := make(chan *os.ProcessState, 1)
	go func() {
		defer close(waitChan)

		state, _ := process.Wait()
		waitChan <- state
	}()

	_ = process.Signal(syscall.SIGTERM)

	select {
	case <-time.After(30 * time.Second):
		fmt.Println("timeout kill")
		if err = process.Kill(); err != nil {
			return nil
		}
		return nil
	case <-waitChan:
		return nil
	}
}
