package initd

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/initd/connecthandler"
	"github.com/baepo-cloud/baepo-node/internal/vsock"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/v1connect"
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
