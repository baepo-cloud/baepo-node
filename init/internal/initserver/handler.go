package initserver

import (
	"context"
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/baepo-cloud/baepo-node/core/vsock"
	"github.com/baepo-cloud/baepo-node/init/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"net/http"
)

type InitServiceServer struct {
	containerService types.ContainerService
	logService       types.LogService
}

var _ nodev1pbconnect.InitHandler = (*InitServiceServer)(nil)

func New(containerService types.ContainerService, logService types.LogService) *InitServiceServer {
	return &InitServiceServer{
		containerService: containerService,
		logService:       logService,
	}
}

func (s InitServiceServer) Start() error {
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
