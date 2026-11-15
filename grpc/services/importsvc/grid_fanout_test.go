package importsvc

import (
	"context"
	"testing"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/grpc/services/importsvc/importer"
	"github.com/srlmgr/backend/grpc/services/importsvc/points"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
)

//nolint:funlen // builds the race/grid policy settings fixture
func TestWithOfftrackHeuristicByRace_UsesRaceGridSettings(t *testing.T) {
	t.Parallel()

	epi := &processor.EventProcInfo{
		Races: []*models.Race{
			{ID: 10, SequenceNo: 1},
			{ID: 20, SequenceNo: 2},
		},
		Grids: []*models.RaceGrid{
			{ID: 101, RaceID: 10, SequenceNo: 1},
			{ID: 201, RaceID: 20, SequenceNo: 1},
		},
		PointSystemSettings: &points.PointSystemSettings{
			Races: []points.RaceSettings{
				{
					PenaltySettings: []points.PointPenaltySettings{
						{Arguments: map[points.PointPolicyType]any{
							points.PointPolicyOfftracksExceeded: points.OfftracksExceededSettings{
								ApplyHeuristic: true,
							},
						}},
					},
				},
				{
					PenaltySettings: []points.PointPenaltySettings{
						{Arguments: map[points.PointPolicyType]any{
							points.PointPolicyOfftracksExceeded: points.OfftracksExceededSettings{
								ApplyHeuristic: false,
							},
						}},
					},
				},
			},
		},
	}

	ctx, err := withOfftrackHeuristicByRace(context.Background(), epi, 101)
	if err != nil {
		t.Fatalf("withOfftrackHeuristicByRace returned unexpected error: %v", err)
	}

	for raceSequenceNo, want := range map[int]bool{
		0: true,
		1: true,
		2: false,
	} {
		if got := importer.OfftrackHeuristicEnabled(ctx, raceSequenceNo); got != want {
			t.Errorf(
				"unexpected heuristic setting for race sequence %d: got %t want %t",
				raceSequenceNo,
				got,
				want,
			)
		}
	}
}
