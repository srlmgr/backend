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

type (
	offtrackHeuristicContextKey struct{}
	skipLapDataCollectionKey    struct{}
)

const (
	ImportDataQuali = "quali" // use for quali only data
	ImportDataRace  = "race"  // use for race only data
	ImportDataAll   = "all"   // use if data contains quali+race
)

// WithOfftrackHeuristicByRace adds per-race offtrack heuristic settings to ctx.
// Race sequence zero applies to single-race imports; multi-race imports use their
// 1-based race sequence numbers.
//
//nolint:whitespace // editor/linter issue
func WithOfftrackHeuristicByRace(
	ctx context.Context,
	enabledByRace map[int]bool,
) context.Context {
	settings := make(map[int]bool, len(enabledByRace))
	for raceSequenceNo, enabled := range enabledByRace {
		settings[raceSequenceNo] = enabled
	}
	return context.WithValue(ctx, offtrackHeuristicContextKey{}, settings)
}

// OfftrackHeuristicEnabled reports whether the heuristic is enabled for a race.
func OfftrackHeuristicEnabled(ctx context.Context, raceSequenceNo int) bool {
	settings, ok := ctx.Value(offtrackHeuristicContextKey{}).(map[int]bool)
	return ok && settings[raceSequenceNo]
}

// WithoutLapDataCollection disables fetching per-lap data while reprocessing an import.
func WithoutLapDataCollection(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipLapDataCollectionKey{}, true)
}

// ShouldCollectLapData reports whether import processing should fetch per-lap data.
func ShouldCollectLapData(ctx context.Context) bool {
	skip, _ := ctx.Value(skipLapDataCollectionKey{}).(bool)
	return !skip
}

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
