package types

import (
	"context"
	"github.com/baepo-cloud/baepo-proto/go/baepo/node/v1/nodev1pbconnect"
)

type (
	RuntimeCreateOptions struct {
		MachineID        string
		Spec             MachineSpec
		Volumes          []*MachineVolume
		NetworkInterface NetworkInterface
	}

	RuntimeProvider interface {
		GC(ctx context.Context, getExpectedMachineIDs func() []string) error

		Create(ctx context.Context, opts RuntimeCreateOptions) (int, error)

		Boot(ctx context.Context, machineID string) error

		Terminate(ctx context.Context, machineID string) error

		NewInitClient(machineID string) (nodev1pbconnect.InitClient, func())
	}
)
