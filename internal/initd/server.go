package initd

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/initd/connecthandler"
	"github.com/baepo-cloud/baepo-node/internal/vsock"
	"github.com/baepo-cloud/baepo-proto/go/baepo/initd/v1/initdv1pbconnect"
	"net/http"
)

func (d *initd) StartServer() error {
	ln, err := vsock.Listen(context.Background(), ServerPort)
	if err != nil {
		return fmt.Errorf("failed to listen on vsock: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(initdv1pbconnect.NewInitDHandler(connecthandler.NewInitDServiceServer(d)))
	server := &http.Server{Handler: mux}
	return server.Serve(ln)
}
