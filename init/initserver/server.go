package initserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/baepo-cloud/baepo-node/init/initserver/connecthandler"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/vsock"
	"github.com/baepo-cloud/baepo-proto/go/baepo/initd/v1/initdv1pbconnect"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	config  types.InitConfig
	logger  *slog.Logger
	errChan chan error
}

var _ types.InitService = (*Server)(nil)

const ServerPort = 9000

func New(config types.InitConfig) *Server {
	return &Server{
		config:  config,
		logger:  slog.Default(),
		errChan: make(chan error, 1),
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
			//case sig := <-sigChan:
			//	_ = init.cmd.Process.Signal(sig)
		}
		time.Sleep(10 * time.Second)
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
	ln, err := vsock.Listen(context.Background(), ServerPort)
	if err != nil {
		return fmt.Errorf("failed to listen on vsock: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(initdv1pbconnect.NewInitDHandler(connecthandler.NewInitDServiceServer(s)))
	server := &http.Server{Handler: mux}
	return server.Serve(ln)
}

func (s *Server) startContainer(config types.InitContainerConfig) {
	jsonConfig, err := json.Marshal(config)
	if err != nil {
		s.errChan <- err
		return
	}

	cmd := exec.Command("/initcontainer", string(jsonConfig))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		s.errChan <- err
		return
	}
}
