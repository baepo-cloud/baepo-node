package containerservice

import (
	"encoding/json"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"os"
	"os/exec"
)

type (
	Service struct {
		logService types.LogService
		containers map[string]*Container
	}

	Container struct {
		cmd    *exec.Cmd
		config coretypes.InitContainerConfig
	}
)

var _ types.ContainerService = (*Service)(nil)

func New(logService types.LogService) *Service {
	return &Service{
		logService: logService,
		containers: map[string]*Container{},
	}
}

func (s *Service) StartContainer(config coretypes.InitContainerConfig) error {
	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return err
	}

	ctr := &Container{
		cmd:    exec.Command("/initcontainer", string(jsonConfig)),
		config: config,
	}
	ctr.cmd.Stdin = os.Stdin
	ctr.cmd.Stdout, ctr.cmd.Stderr = s.logService.NewContainerLogWriter(config)
	if err = ctr.cmd.Start(); err != nil {
		return err
	}

	s.containers[config.Name] = ctr
	return nil
}

func (s *Service) ContainersState() []*types.ContainerState {
	//TODO implement me
	panic("implement me")
}
