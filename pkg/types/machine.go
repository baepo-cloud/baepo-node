package types

import (
	"context"
	"errors"
	"time"
)

type (
	MachineCreateOptions struct {
		Name     *string
		Vcpus    int
		Memory   int64
		Timeout  *int
		Metadata map[string]string
		Env      map[string]string
	}

	MachineState string

	Machine struct {
		ID                string
		Name              *string
		State             MachineState
		NodeID            *string
		Vcpus             int
		Memory            int64
		Timeout           *int
		StartedAt         *time.Time
		ExpiresAt         *time.Time
		TerminatedAt      *time.Time
		TerminationReason *string
		Metadata          MapStringString
		Env               MapStringString
		CreatedAt         time.Time
		UpdatedAt         time.Time
	}

	MachineRuntime struct {
		MachineID    string
		State        MachineState
		NodeID       string
		PID          *int
		TapInterface *string
		MacAddress   *string
		IPAddress    *string
	}

	MachineStore interface {
		FindMachineByID(ctx context.Context, id string) (*Machine, error)

		CreateMachine(ctx context.Context, machine *Machine) error

		UpdateMachine(ctx context.Context, machine *Machine) error

		UpdateMachineState(ctx context.Context, machineID string, state MachineState) error

		EtcdMachineRuntimeBasePrefix() string

		EtcdMachineRuntimePrefix(machineID string) string

		SaveMachineRuntime(ctx context.Context, runtime *MachineRuntime) error

		FindMachineRuntime(ctx context.Context, machineID string) (*MachineRuntime, error)

		DeleteMachineRuntime(ctx context.Context, runtime *MachineRuntime) error
	}

	MachineService interface {
		Create(ctx context.Context, opts MachineCreateOptions) (*Machine, error)

		RequestTermination(ctx context.Context, machineID string) error
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
	ErrMachineNotFound        = errors.New("machine not found")
	ErrMachineRuntimeNotFound = errors.New("machine runtime not found")
)
