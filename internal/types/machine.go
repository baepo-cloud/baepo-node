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
		Image      string
		Vcpus      uint32
		MemoryMB   uint64
		Env        map[string]string
		User       string
		WorkingDir string
		Command    []string
	}
)

const (
	MachineStatusScheduling  MachineStatus = "scheduling"
	MachineStatusStarting    MachineStatus = "starting"
	MachineStatusRunning     MachineStatus = "running"
	MachineStatusTerminating MachineStatus = "terminating"
	MachineStatusTerminated  MachineStatus = "terminated"
)

var ErrMachineNotFound = errors.New("machine not found")
