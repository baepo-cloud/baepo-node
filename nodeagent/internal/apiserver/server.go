package apiserver

import (
	"connectrpc.com/connect"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
	"github.com/expected-so/canonicallog"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Server struct {
	service         types.NodeService
	runtimeProvider types.RuntimeProvider
	config          *types.Config
	httpServer      *http.Server
}

var _ nodev1pbconnect.NodeServiceHandler = (*Server)(nil)

func New(service types.NodeService, runtimeProvider types.RuntimeProvider, config *types.Config) *Server {
	return &Server{
		service:         service,
		runtimeProvider: runtimeProvider,
		config:          config,
	}
}

func (s *Server) Start(ctx context.Context) error {
	slog.Info("starting api server", slog.String("addr", s.config.APIAddr))

	mux := http.NewServeMux()
	mux.Handle(nodev1pbconnect.NewNodeServiceHandler(s))

	s.httpServer = &http.Server{
		Addr:    s.config.APIAddr,
		Handler: mux,
	}

	unixSocket := filepath.Join(s.config.StorageDirectory, "agent.sock")
	_ = os.Remove(unixSocket)
	unixListener, err := net.Listen("unix", unixSocket)
	if err != nil {
		return fmt.Errorf("failed to setup unix socket: %w", err)
	}

	tcpListener, err := tls.Listen("tcp", s.httpServer.Addr, &tls.Config{
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			config := &tls.Config{
				ClientAuth: tls.RequireAndVerifyClientCert,
				MinVersion: tls.VersionTLS12,
			}
			if cert := s.service.TLSCertificate(); cert != nil {
				config.Certificates = []tls.Certificate{*cert}
			}
			if cert := s.service.AuthorityCertificate(); cert != nil {
				config.ClientCAs = x509.NewCertPool()
				config.ClientCAs.AddCert(s.service.AuthorityCertificate())
			}
			return config, nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to setup listener for api server: %w", err)
	}

	go s.httpServer.Serve(tcpListener)
	go s.httpServer.Serve(unixListener)

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	slog.Info("shutting down api server")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) newLoggerUnaryInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			logContext := canonicallog.NewLogLine(ctx)
			startedAt := time.Now()
			canonicallog.LogAttr(logContext, slog.String("procedure", request.Spec().Procedure))

			res, err := next(logContext, request)

			canonicallog.LogDuration(logContext, time.Now().Sub(startedAt))
			canonicallog.PrintLine(logContext, "api-request")

			return res, err
		}
	}
}
