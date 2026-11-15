package importsvc

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/grpc/services/importsvc/importer"
	"github.com/srlmgr/backend/grpc/services/importsvc/points"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
)

//nolint:whitespace // editor/linter issue
func withOfftrackHeuristicByRace(
	ctx context.Context,
	epi *processor.EventProcInfo,
	anchorGridID int32,
) (
	context.Context,
	error,
) {
	if epi.PointSystemSettings == nil {
		return importer.WithOfftrackHeuristicByRace(ctx, nil), nil
	}

	enabledByRace := make(map[int]bool, len(epi.Races)+1)
	enabled, err := offtrackHeuristicEnabledForGrid(ctx, epi, anchorGridID)
	if err != nil {
		return nil, err
	}
	enabledByRace[0] = enabled

	for _, race := range epi.Races {
		grid, ok := lo.Find(epi.Grids, func(grid *models.RaceGrid) bool {
			return grid.RaceID == race.ID
		})
		if !ok {
			// resolveTargetGridIDs reports this if the import contains the race.
			continue
		}
		enabled, err := offtrackHeuristicEnabledForGrid(ctx, epi, grid.ID)
		if err != nil {
			return nil, err
		}
		enabledByRace[int(race.SequenceNo)] = enabled
	}

	return importer.WithOfftrackHeuristicByRace(ctx, enabledByRace), nil
}

//nolint:whitespace // editor/linter issue
func offtrackHeuristicEnabledForGrid(
	ctx context.Context,
	epi *processor.EventProcInfo,
	gridID int32,
) (
	bool,
	error,
) {
	raceNo, gridNo, err := epi.ResolverFunc(ctx)(gridID)
	if err != nil {
		return false, err
	}
	if epi.PointSystemSettings == nil || raceNo < 0 {
		return false, nil
	}
	if int(raceNo) >= len(epi.PointSystemSettings.Races) {
		return false, nil
	}

	raceSettings := epi.PointSystemSettings.Races[raceNo]
	if gridNo < 0 || int(gridNo) >= len(raceSettings.PenaltySettings) {
		return false, nil
	}
	settings, ok := raceSettings.PenaltySettings[gridNo].
		Arguments[points.PointPolicyOfftracksExceeded].(points.OfftracksExceededSettings)
	return ok && settings.ApplyHeuristic, nil
}

// processImportPayload processes payload via importProcessor, returning one
// ParsedImportPayload per detected race. For processors that support heat-race
// detection (MultiRaceProcessor), all races found in payload are returned; otherwise a
// single-element slice built from the standard Process result is returned.
//
//nolint:whitespace // editor/linter issue
func processImportPayload(
	ctx context.Context,
	importProcessor importer.ProcessImport,
	format importer.ImportFormat,
	payload []byte,
) ([]*importer.ParsedImportPayload, error) {
	if multiRace, ok := importer.SupportsMultiRace(importProcessor); ok {
		return multiRace.ProcessMultiRace(ctx, format, payload)
	}

	parsed, err := importProcessor.Process(ctx, format, payload)
	if err != nil {
		return nil, err
	}

	return []*importer.ParsedImportPayload{parsed}, nil
}

// selectInputForGrid picks the parsed input whose resolved target grid matches
// gridID. This is used when reprocessing a stored import batch (e.g.
// ResolveMappings), where the same multi-race payload must be re-split and only
// the entry for this specific grid used.
//
//nolint:whitespace // editor/linter issue
func selectInputForGrid(
	epi *processor.EventProcInfo,
	inputs []*importer.ParsedImportPayload,
	gridID int32,
) (*importer.ParsedImportPayload, error) {
	targetGridIDs, err := resolveTargetGridIDs(epi, inputs, gridID)
	if err != nil {
		return nil, err
	}

	for i, targetGridID := range targetGridIDs {
		if targetGridID == gridID {
			return inputs[i], nil
		}
	}

	return nil, fmt.Errorf("%w: race grid %d", processor.ErrGridNotFound, gridID)
}

// resolveTargetGridIDs maps each parsed input to the race grid its results belong to.
// For a single, non-heat input it simply targets anchorGridID (existing behavior). For
// multiple inputs (heat races), each input's RaceSequenceNo is matched against the
// event's races (already loaded on epi) to find the corresponding grid.
//
//nolint:whitespace // editor/linter issue
func resolveTargetGridIDs(
	epi *processor.EventProcInfo,
	inputs []*importer.ParsedImportPayload,
	anchorGridID int32,
) ([]int32, error) {
	if len(inputs) == 1 && inputs[0].RaceSequenceNo == 0 {
		return []int32{anchorGridID}, nil
	}

	gridIDs := make([]int32, len(inputs))
	for i, input := range inputs {
		race, ok := lo.Find(epi.Races, func(r *models.Race) bool {
			return r.SequenceNo == int32(input.RaceSequenceNo)
		})
		if !ok {
			return nil, fmt.Errorf(
				"%w: race sequence %d", processor.ErrRaceNotFound, input.RaceSequenceNo,
			)
		}

		grid, ok := lo.Find(epi.Grids, func(g *models.RaceGrid) bool {
			return g.RaceID == race.ID
		})
		if !ok {
			return nil, fmt.Errorf("%w: race %d", processor.ErrGridNotFound, race.ID)
		}

		gridIDs[i] = grid.ID
	}

	return gridIDs, nil
}
