package nodeservice

import (
	"context"
	"github.com/baepo-app/baepo-node/internal/types"
	"log/slog"
)

func (s *Service) StopMachine(ctx context.Context, machineID string) (*types.Machine, error) {
	machine, err := s.FindMachine(ctx, machineID)
	if err != nil {
		return nil, err
	}

	slog.Info("stopping machine", slog.String("machine-id", machineID))

	if machine.RuntimePID > 0 {
		err = s.runtimeProvider.Terminate(ctx, machine)
		if err != nil {
			slog.Error("failed to terminate machine",
				slog.String("machine-id", machineID),
				slog.Any("error", err))
		}
	}

	if machine.NetworkInterface != nil {
		err = s.networkProvider.ReleaseInterface(machine.NetworkInterface.Name)
		if err != nil {
			slog.Error("failed to release machine network",
				slog.String("machine-id", machineID),
				slog.Any("error", err))
		}
	}

	if machine.Volume != nil {
		_ = s.volumeProvider.DeleteVolume(ctx, machine.Volume)
		if err != nil {
			slog.Error("failed to delete machine volume",
				slog.String("machine-id", machineID),
				slog.Any("error", err))
		}
	}

	s.lock.Lock()
	delete(s.machines, machineID)
	s.lock.Unlock()

	return machine, nil
}
