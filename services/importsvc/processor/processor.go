// Package processor defines simulation-specific import processors.
package processor

import (
	"context"
	"slices"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
)

// ImportFormat is the persisted format value used by processors.
type ImportFormat = string

// ProcessImport describes a simulation-specific import processor.
type ProcessImport interface {
	Process(
		ctx context.Context,
		format ImportFormat,
		payload any,
	) ([]*commonv1.ResultEntry, []*commonv1.UnresolvedMapping, error)
}

// FormatSupporter can be implemented by processors that expose supported formats.
type FormatSupporter interface {
	SupportedFormats() []ImportFormat
}

// SupportsFormat reports whether the processor supports the given import format.
func SupportsFormat(processor ProcessImport, format ImportFormat) bool {
	supporter, ok := processor.(FormatSupporter)
	if !ok {
		return false
	}

	return slices.Contains(supporter.SupportedFormats(), format)
}
