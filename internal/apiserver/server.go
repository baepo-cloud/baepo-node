package apiserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/pkg/proto/baepo/node/v1/v1connect"
	"log/slog"
	"net/http"
)

type Server struct {
	service    types.NodeService
	config     *types.NodeServerConfig
	httpServer *http.Server
}

var _ v1connect.NodeServiceHandler = (*Server)(nil)

func New(service types.NodeService, config *types.NodeServerConfig) *Server {
	return &Server{
		service: service,
		config:  config,
	}
}

func (s *Server) Start(ctx context.Context) error {
	slog.Info("starting api server", slog.String("addr", s.config.APIAddr))

	mux := http.NewServeMux()
	mux.Handle(v1connect.NewNodeServiceHandler(s))

	s.httpServer = &http.Server{
		Addr:    s.config.APIAddr,
		Handler: mux,
		TLSConfig: &tls.Config{
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
		},
	}

	lis, err := tls.Listen("tcp", s.httpServer.Addr, s.httpServer.TLSConfig)
	if err != nil {
		return fmt.Errorf("failed to setup listener for api server: %w", err)
	}

	go s.httpServer.Serve(lis)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	slog.Info("shutting down api server")
	return s.httpServer.Shutdown(ctx)
}
