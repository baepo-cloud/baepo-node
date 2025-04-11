package nodeservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"gorm.io/gorm"
)

func (s *Service) FindMachine(ctx context.Context, machineID string) (*types.Machine, error) {
	var machine *types.Machine
	err := s.db.WithContext(ctx).
		Joins("Volume").
		Joins("NetworkInterface").
		First(&machine, "id = ?", machineID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, types.ErrMachineNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to find machine: %w", err)
	}

	return machine, nil
}
