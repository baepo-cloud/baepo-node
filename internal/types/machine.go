package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type (
	MachineState string

	MachineDesiredState string

	Machine struct {
		ID               string `gorm:"primaryKey"`
		State            MachineState
		DesiredState     MachineDesiredState
		RuntimePID       *int `gorm:"column:runtime_pid"`
		Spec             *MachineSpec
		Volume           *Volume
		NetworkInterface *NetworkInterface
		CreatedAt        time.Time
		TerminatedAt     *time.Time
	}

	MachineSpec struct {
		Image      string
		Cpus       uint32
		MemoryMB   uint64
		Env        map[string]string
		User       string
		WorkingDir string
		Command    []string
	}
)

const (
	MachineStatePending     MachineState = "pending"
	MachineStateStarting    MachineState = "starting"
	MachineStateRunning     MachineState = "running"
	MachineStateDegraded    MachineState = "degraded"
	MachineStateError       MachineState = "error"
	MachineStateTerminating MachineState = "terminating"
	MachineStateTerminated  MachineState = "terminated"

	MachineDesiredStatePending    MachineDesiredState = "pending"
	MachineDesiredStateRunning    MachineDesiredState = "running"
	MachineDesiredStateTerminated MachineDesiredState = "terminated"
)

var ErrMachineNotFound = errors.New("machine not found")

func (*MachineSpec) GormDataType() string {
	return "jsonb"
}

func (s *MachineSpec) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &s)
}

func (s *MachineSpec) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s MachineState) MatchDesiredState(desired MachineDesiredState) bool {
	switch s {
	case MachineStatePending:
		return desired == MachineDesiredStatePending
	case MachineStateRunning, MachineStateDegraded:
		return desired == MachineDesiredStateRunning
	case MachineStateTerminated:
		return desired == MachineDesiredStateTerminated
	default:
		return false
	}
}
