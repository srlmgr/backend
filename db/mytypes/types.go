package mytypes

import (
	"database/sql/driver"
	"encoding/json"
)

type (
	SourceType   string
	TargetType   string
	ImportFormat string
	TeamDrivers  struct {
		DriverIDs []int32 `json:"driverIDs"`
	}
)

func (s *SourceType) Scan(v any) error {
	arg, ok := v.(string)
	if !ok {
		return nil
	}
	*s = SourceType(arg)
	return nil
}

func (s SourceType) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *TargetType) Scan(v any) error {
	arg, ok := v.(string)
	if !ok {
		return nil
	}
	*s = TargetType(arg)
	return nil
}

func (s TargetType) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *ImportFormat) Scan(v any) error {
	arg, ok := v.(string)
	if !ok {
		return nil
	}
	*s = ImportFormat(arg)
	return nil
}

func (s ImportFormat) Value() (driver.Value, error) {
	return string(s), nil
}

func (t *TeamDrivers) Scan(v any) error {
	bytes, ok := v.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, t)
}

func (t TeamDrivers) Value() (driver.Value, error) {
	return json.Marshal(t)
}
