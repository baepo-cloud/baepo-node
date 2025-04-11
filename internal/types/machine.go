package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type (
	MachineStatus string

	Machine struct {
		ID               string `gorm:"primaryKey"`
		Status           MachineStatus
		RuntimePID       *int `gorm:"column:runtime_pid"`
		Spec             *MachineSpec
		Volume           *Volume
		NetworkInterface *NetworkInterface
		CreatedAt        time.Time
		TerminatedAt     *time.Time
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
	MachineStatusStarting    MachineStatus = "starting"
	MachineStatusRunning     MachineStatus = "running"
	MachineStatusTerminating MachineStatus = "terminating"
	MachineStatusTerminated  MachineStatus = "terminated"
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
