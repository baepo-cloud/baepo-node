package runtimeservice

import (
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"path"
)

type Service struct {
	config *types.Config
}

var _ types.RuntimeService = (*Service)(nil)

func New(config *types.Config) *Service {
	return &Service{
		config: config,
	}
}

func (s *Service) getRuntimeDir(machineID string) string {
	return path.Join(s.config.StorageDirectory, "runtimes", machineID)
}

func (s *Service) getRuntimeConfigPath(machineID string) string {
	return path.Join(s.getRuntimeDir(machineID), "runtimeconfig.json")
}
