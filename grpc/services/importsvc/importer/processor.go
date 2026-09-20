// package importer defines simulation-specific import processors.
package importer

import (
	"context"
	"slices"
)

// ImportFormat is the persisted format value used by processors.
type (
	ImportFormat = string
	ImportData   = string
)

const (
	ImportDataQuali = "quali" // use for quali only data
	ImportDataRace  = "race"  // use for race only data
	ImportDataAll   = "all"   // use if data contains quali+race
)

// ProcessImport describes a simulation-specific import processor.
type ProcessImport interface {
	Process(
		ctx context.Context,
		format ImportFormat,
		payload any,
	) (*ParsedImportPayload, error)
	// Combine(
	// 	ctx context.Context,
	// 	quali, race *ParsedImportPayload) (*ParsedImportPayload, error)
}

// FormatSupporter can be implemented by processors that expose supported formats.
type FormatSupporter interface {
	SupportedFormats() []ImportFormat
}

// MultiRaceProcessor can be implemented by processors that can detect multiple races
// (e.g. heat races) within a single payload and return one ParsedImportPayload per
// race.
type MultiRaceProcessor interface {
	ProcessMultiRace(
		ctx context.Context,
		format ImportFormat,
		payload any,
	) ([]*ParsedImportPayload, error)
}

// SupportsFormat reports whether the processor supports the given import format.
func SupportsFormat(processor ProcessImport, format ImportFormat) bool {
	supporter, ok := processor.(FormatSupporter)
	if !ok {
		return false
	}

	return slices.Contains(supporter.SupportedFormats(), format)
}

// SupportsMultiRace returns the MultiRaceProcessor implementation of processor, if any.
func SupportsMultiRace(processor ProcessImport) (MultiRaceProcessor, bool) {
	multiRace, ok := processor.(MultiRaceProcessor)
	return multiRace, ok
}
