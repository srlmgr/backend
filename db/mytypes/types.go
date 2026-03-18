package mytypes

import "database/sql/driver"

type (
	SourceType   string
	TargetType   string
	ImportFormat string
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
