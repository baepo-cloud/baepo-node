package gatewayserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"log/slog"
	"net/http"
)

type Server struct {
	registrationService types.RegistrationService
	machineService      types.MachineService
	config              *types.Config
	httpServer          *http.Server
}

func New(
	registrationService types.RegistrationService,
	machineService types.MachineService,
	config *types.Config,
) *Server {
	return &Server{
		registrationService: registrationService,
		machineService:      machineService,
		config:              config,
	}
}

func (s *Server) Start(ctx context.Context) error {
	slog.Info("starting gateway server", slog.String("addr", s.config.GatewayAddr))

	s.httpServer = &http.Server{
		Addr:    s.config.GatewayAddr,
		Handler: s.Handler(),
		TLSConfig: &tls.Config{
			GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
				config := &tls.Config{
					ClientAuth: tls.RequireAndVerifyClientCert,
					MinVersion: tls.VersionTLS12,
				}
				if cert := s.registrationService.TLSCertificate(); cert != nil {
					config.Certificates = []tls.Certificate{*cert}
				}
				if cert := s.registrationService.AuthorityCertificate(); cert != nil {
					config.ClientCAs = x509.NewCertPool()
					config.ClientCAs.AddCert(cert)
				}
				return config, nil
			},
		},
	}

	lis, err := tls.Listen("tcp", s.httpServer.Addr, s.httpServer.TLSConfig)
	if err != nil {
		return fmt.Errorf("failed to setup listener for gateway server: %w", err)
	}

	go s.httpServer.Serve(lis)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	slog.Info("shutting down gateway server")
	return s.httpServer.Shutdown(ctx)
}
