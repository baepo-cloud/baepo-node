package initserver

import (
	"context"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/vsock"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"log/slog"
	"net/http"
)

type InitServiceServer struct {
	log              *slog.Logger
	containerService types.ContainerService
	logService       types.LogService
}

var _ nodev1pbconnect.InitHandler = (*InitServiceServer)(nil)

func New(containerService types.ContainerService, logService types.LogService) *InitServiceServer {
	return &InitServiceServer{
		log:              slog.With(slog.String("component", "server")),
		containerService: containerService,
		logService:       logService,
	}
}

func (s InitServiceServer) Start() error {
	s.log.Info("starting server")

	ln, err := vsock.Listen(context.Background(), coretypes.InitServerPort)
	if err != nil {
		return fmt.Errorf("failed to listen on vsock: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(nodev1pbconnect.NewInitHandler(s))
	server := &http.Server{
		Handler: mux,
	}
	return server.Serve(ln)
}
