package nodeservice

import (
	"context"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"log/slog"
	"time"
)

func (s *Service) StopMachine(ctx context.Context, machineID string) (machine *types.Machine, err error) {
	startedAt := time.Now()
	log := slog.With(slog.String("machine-id", machineID))
	log.Info("stopping machine")

	defer func() {
		log = log.With(slog.Duration("duration", time.Now().Sub(startedAt)))
		if err != nil {
			log.Error("failed to stop machine", slog.Any("error", err))
		} else {
			log.Info("machine stopped")
		}
	}()

	machine, err = s.FindMachine(ctx, machineID)
	if err != nil {
		return nil, err
	}

	if machine.RuntimePID != nil && *machine.RuntimePID > 0 {
		log.Info("terminating machine runtime")
		err = s.runtimeProvider.Terminate(ctx, machine)
		if err != nil {
			return nil, err
		}
	}

	if machine.NetworkInterface != nil {
		log.Info("release machine network interface")
		err = s.networkProvider.ReleaseInterface(ctx, machine.NetworkInterface.Name)
		if err != nil {
			return nil, err
		}
	}

	if machine.Volume != nil {
		log.Info("deleting machine volume")
		_ = s.volumeProvider.DeleteVolume(ctx, machine.Volume)
		if err != nil {
			return nil, err
		}
	}
	return machine, nil
}
