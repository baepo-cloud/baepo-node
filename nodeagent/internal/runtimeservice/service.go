package runtimeservice

import (
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"log/slog"
	"path"
)

type Service struct {
	config *types.Config
	log    *slog.Logger
}

var _ types.RuntimeService = (*Service)(nil)

func New(config *types.Config) *Service {
	return &Service{
		config: config,
		log:    slog.With(slog.String("component", "runtimeservice")),
	}
}

func (s *Service) getRuntimeDir(machineID string) string {
	return path.Join(s.config.StorageDirectory, "runtimes", machineID)
}

func (s *Service) getRuntimeConfigPath(machineID string) string {
	return path.Join(s.getRuntimeDir(machineID), "runtimeconfig.json")
}
