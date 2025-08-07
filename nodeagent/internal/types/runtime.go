package types

import (
	"context"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
)

type (
	RuntimeStartOptions struct {
		Machine *Machine
	}

	RuntimeService interface {
		Start(ctx context.Context, opts RuntimeStartOptions) error

		Terminate(ctx context.Context, machineID string) error

		GetClient(machineID string) (nodev1pbconnect.RuntimeClient, func())

		GetMachineDirectory(machineID string) string
	}
)
