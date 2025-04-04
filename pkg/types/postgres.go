package types

import (
	"database/sql/driver"
	"encoding/json"
)

type MapStringString map[string]string

func (*MapStringString) GormDataType() string {
	return "jsonb"
}

func (v *MapStringString) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &v)
}

func (v *MapStringString) Value() (driver.Value, error) {
	return json.Marshal(v)
}
