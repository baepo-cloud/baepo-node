package types

import "context"

type RuntimeProvider interface {
	Create(ctx context.Context, machine *Machine) (int, error)

	Boot(ctx context.Context, machine *Machine) error

	Terminate(ctx context.Context, machine *Machine) error

	RecoverRunningMachines(ctx context.Context) ([]*Machine, error)
}
