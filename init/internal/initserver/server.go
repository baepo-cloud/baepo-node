package initserver

import (
	"context"
	"encoding/json"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/vsock"
	"github.com/baepo-cloud/baepo-node/init/internal/connecthandler"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

type (
	Server struct {
		config     coretypes.InitConfig
		logger     *slog.Logger
		errChan    chan error
		containers map[string]*Container
		logService types.LogService
	}

	Container struct {
		cmd    *exec.Cmd
		config coretypes.InitContainerConfig
	}
)

var _ types.InitService = (*Server)(nil)

func New(logService types.LogService, config coretypes.InitConfig) *Server {
	return &Server{
		config:     config,
		logger:     slog.Default(),
		errChan:    make(chan error, 1),
		containers: map[string]*Container{},
		logService: logService,
	}
}

func (s *Server) Run() error {
	slog.Info("starting init")
	if err := s.mountFilesystems(); err != nil {
		panic(err)
	}

	if err := s.setupNetwork(); err != nil {
		return fmt.Errorf("failed to setup network: %v", err)
	}

	for _, containerConfig := range s.config.Containers {
		go s.startContainer(containerConfig)
	}

	go func() {
		s.errChan <- s.startHttpServer()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var err error
	running := true
	for running {
		select {
		case err = <-s.errChan:
			running = false
		case sig := <-sigChan:
			for _, ctr := range s.containers {
				_ = ctr.cmd.Process.Signal(sig)
			}
		}
	}

	slog.Error("shutting down", slog.Any("error", err))
	syscall.Sync()
	_ = syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	return nil
}

func (s *Server) ContainersState() []*types.InitContainerState {
	//TODO implement me
	panic("implement me")
}

func (s *Server) startHttpServer() error {
	ln, err := vsock.Listen(context.Background(), coretypes.InitServerPort)
	if err != nil {
		return fmt.Errorf("failed to listen on vsock: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(nodev1pbconnect.NewInitHandler(connecthandler.NewInitServiceServer(s, s.logService)))
	server := &http.Server{Handler: mux}
	return server.Serve(ln)
}

func (s *Server) startContainer(config coretypes.InitContainerConfig) {
	jsonConfig, err := json.Marshal(config)
	if err != nil {
		s.errChan <- err
		return
	}

	ctr := &Container{
		cmd:    exec.Command("/initcontainer", string(jsonConfig)),
		config: config,
	}
	ctr.cmd.Stdin = os.Stdin
	ctr.cmd.Stdout, ctr.cmd.Stderr = s.logService.NewContainerLogWriter(config)
	if err = ctr.cmd.Start(); err != nil {
		s.errChan <- err
		return
	}

	s.containers[config.Name] = ctr
}
