package types

import "context"

type (
	RuntimeCreateOptions struct {
		MachineID        string
		Spec             MachineSpec
		Volume           Volume
		NetworkInterface NetworkInterface
	}

	RuntimeProvider interface {
		Create(ctx context.Context, opts RuntimeCreateOptions) (int, error)

		Boot(ctx context.Context, machineID string) error

		Terminate(ctx context.Context, machineID string) error

		Healthcheck(ctx context.Context, machineID string) error
	}
)
