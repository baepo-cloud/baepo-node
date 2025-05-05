package types

import (
	"context"
	"crypto/tls"
	"crypto/x509"
)

type (
	NodeCreateMachineOptions struct {
		MachineID    string
		DesiredState MachineDesiredState
		Spec         MachineSpec
	}

	NodeUpdateMachineDesiredStateOptions struct {
		MachineID    string
		DesiredState MachineDesiredState
	}

	NodeService interface {
		Start(ctx context.Context) error

		Stop(ctx context.Context) error

		AuthorityCertificate() *x509.Certificate

		TLSCertificate() *tls.Certificate

		ListMachines(ctx context.Context) ([]*Machine, error)

		FindMachine(ctx context.Context, machineID string) (*Machine, error)

		CreateMachine(ctx context.Context, opts NodeCreateMachineOptions) (*Machine, error)

		UpdateMachineDesiredState(ctx context.Context, opts NodeUpdateMachineDesiredStateOptions) (*Machine, error)
	}
)
