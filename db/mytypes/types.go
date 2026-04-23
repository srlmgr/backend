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
	RaceSimImportFormat struct {
		Format               ImportFormat `json:"format"`
		AllowMultipleUploads string       `json:"allowMultipleUploads"`
	}
	ImportBatchMeta struct {
		Race  string `json:"race,omitempty"`  // fn in zip for race data
		Quali string `json:"quali,omitempty"` // fn in zip for quali data
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

func (t *RaceSimImportFormat) Scan(v any) error {
	bytes, ok := v.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, t)
}

func (t RaceSimImportFormat) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *ImportBatchMeta) Scan(v any) error {
	bytes, ok := v.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, t)
}

func (t ImportBatchMeta) Value() (driver.Value, error) {
	return json.Marshal(t)
}
