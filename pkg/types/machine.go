package types

import (
	"context"
	"errors"
)

type (
	MachineState string

	Machine struct {
		ID               string
		State            MachineState
		RuntimePID       int
		Spec             *MachineSpec
		Volume           *Volume
		NetworkInterface *NetworkInterface
	}

	MachineSpec struct {
		Vcpus  int
		Memory int64
		Env    map[string]string
	}

	RuntimeProvider interface {
		Create(ctx context.Context, machine *Machine) (int, error)

		Boot(ctx context.Context, machine *Machine) error

		Terminate(ctx context.Context, machine *Machine) error
	}
)

const (
	MachineStateScheduling  MachineState = "scheduling"
	MachineStateStarting    MachineState = "starting"
	MachineStateRunning     MachineState = "running"
	MachineStateTerminating MachineState = "terminating"
	MachineStateTerminated  MachineState = "terminated"
)

var (
	ErrMachineNotFound = errors.New("machine not found")
)
