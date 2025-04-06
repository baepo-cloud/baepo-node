package nodeservice

import (
	"context"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"log/slog"
)

func (s *Service) HealthcheckMachine(ctx context.Context, machineID string) (*types.Machine, error) {
	machine, err := s.FindMachine(ctx, machineID)
	if err != nil {
		return nil, err
	}

	slog.Info("performing machine healthcheck", slog.String("machine-id", machineID))
	err = s.runtimeProvider.Healthcheck(ctx, machine)
	if err != nil {
		return nil, err
	}

	return machine, nil
}
