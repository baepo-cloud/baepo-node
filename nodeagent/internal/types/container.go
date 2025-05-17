package types

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type (
	Container struct {
		ID        string
		MachineID string
		Machine   *Machine
		Spec      *ContainerSpec
		CreatedAt time.Time
	}

	ContainerSpec struct {
		Name        string
		Image       string
		Env         map[string]string
		Command     []string
		Healthcheck *ContainerHealthcheckSpec
		WorkingDir  string
	}

	ContainerHealthcheckSpec struct {
		InitialDelaySeconds int32
		PeriodSeconds       int32
		Http                *ContainerHttpHealthcheckSpec
	}

	ContainerHttpHealthcheckSpec struct {
		Method  string
		Path    string
		Port    int32
		Headers map[string]string
	}
)

func (*ContainerSpec) GormDataType() string {
	return "jsonb"
}

func (s *ContainerSpec) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &s)
}

func (s *ContainerSpec) Value() (driver.Value, error) {
	return json.Marshal(s)
}
