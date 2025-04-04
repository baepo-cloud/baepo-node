package types

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
)

type (
	NodeServerConfig struct {
		IPAddr           string
		ServerAddr       string
		GatewayAddr      string
		StorageDirectory string
	}

	NodeStartMachineOptions struct {
		MachineID string
		Spec      MachineSpec
	}

	NodeService interface {
		Start(ctx context.Context) error

		Stop(ctx context.Context) error

		AuthorityCertificate() *x509.Certificate

		TLSCertificate() *tls.Certificate

		FindMachine(ctx context.Context, machineID string) (*Machine, error)

		HealthcheckMachine(ctx context.Context, machineID string) (*Machine, error)

		StartMachine(ctx context.Context, opts NodeStartMachineOptions) (*Machine, error)

		StopMachine(ctx context.Context, machineID string) (*Machine, error)
	}
)

var (
	ErrNodeMachineNotFound = errors.New("node machine not found")
)
