package types

import (
	"errors"
)

type (
	MachineStatus string

	Machine struct {
		ID               string
		Status           MachineStatus
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
)

const (
	MachineStatusScheduling  MachineStatus = "scheduling"
	MachineStatusStarting    MachineStatus = "starting"
	MachineStatusRunning     MachineStatus = "running"
	MachineStatusTerminating MachineStatus = "terminating"
	MachineStatusTerminated  MachineStatus = "terminated"
)

var (
	ErrMachineNotFound = errors.New("machine not found")
)
