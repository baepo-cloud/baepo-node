package initd

import (
	"context"
	"fmt"
	"github.com/baepo-app/baepo-node/internal/initd/connecthandler"
	"github.com/baepo-app/baepo-node/internal/proto/baepo/node/v1/v1connect"
	"github.com/baepo-app/baepo-node/internal/vsock"
	"net/http"
)

func (d *initd) StartServer() error {
	ln, err := vsock.Listen(context.Background(), ServerPort)
	if err != nil {
		return fmt.Errorf("failed to listen on vsock: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(v1connect.NewInitDHandler(connecthandler.NewInitDServiceServer(d)))
	server := &http.Server{Handler: mux}
	return server.Serve(ln)
}
