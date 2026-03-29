// package importer defines simulation-specific import processors.
package importer

import (
	"context"
	"slices"
)

// ImportFormat is the persisted format value used by processors.
type ImportFormat = string

// ProcessImport describes a simulation-specific import processor.
type ProcessImport interface {
	Process(
		ctx context.Context,
		format ImportFormat,
		payload any,
	) (*ParsedImportPayload, error)
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
