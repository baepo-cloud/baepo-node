package types

import (
	"database/sql/driver"
	"encoding/json"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
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

	ContainerSpec coretypes.ContainerSpec
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
