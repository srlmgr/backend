//nolint:whitespace,funlen // editor/linter issue, test logic is long
package standings

import (
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/aarondl/opt/null"

	"github.com/srlmgr/backend/db/models"
	mytypes "github.com/srlmgr/backend/db/mytypes"
)

type testParticipation struct {
	raceGridID     int32
	referenceID    int32
	finishPosition int32
}

func (p testParticipation) RaceGridID() int32     { return p.raceGridID }
func (p testParticipation) ReferenceID() int32    { return p.referenceID }
func (p testParticipation) FinishPosition() int32 { return p.finishPosition }

func TestComputeCarriesStandingsAcrossMissingEvents(t *testing.T) {
	compute := NewComputeStandings()

	actual := compute.Compute(&ComputeStandingsInput{
		EventIDs: []int32{10, 20, 30},
		Bookings: []*models.BookingEntry{
			bookingEntry(10, 100, 1, 10, "finish_pos"),
			bookingEntry(10, 100, 2, 8, "finish_pos"),
			bookingEntry(10, 100, 2, 1, "fastest_lap"),
			bookingEntry(20, 200, 2, 6, "finish_pos"),
			bookingEntry(30, 300, 1, 7, "finish_pos"),
		},
		Participations: []Participation{
			testParticipation{raceGridID: 100, referenceID: 1, finishPosition: 1},
			testParticipation{raceGridID: 100, referenceID: 2, finishPosition: 2},
			testParticipation{raceGridID: 200, referenceID: 2, finishPosition: 1},
			testParticipation{raceGridID: 300, referenceID: 1, finishPosition: 2},
		},
		ReferenceID: func(booking *models.BookingEntry) int32 {
			return booking.DriverID.GetOrZero()
		},
	})

	if len(actual) != 2 {
		t.Fatalf("expected 2 standings entries, got %d", len(actual))
	}
	assertComputedStanding(t, actual[0], 1, &commonv1.StandingData{
		Position:       1,
		PrevPosition:   2,
		TotalPoints:    17,
		BonusPoints:    0,
		PenaltyPoints:  0,
		NumEvents:      2,
		NumRaces:       2,
		NumPenaltyFree: 0,
		NumWins:        1,
		NumPodiums:     2,
		NumTop5:        2,
		NumTop10:       2,
	})
	assertComputedStanding(t, actual[1], 2, &commonv1.StandingData{
		Position:       2,
		PrevPosition:   1,
		TotalPoints:    15,
		BonusPoints:    1,
		PenaltyPoints:  0,
		NumEvents:      2,
		NumRaces:       2,
		NumPenaltyFree: 0,
		NumWins:        1,
		NumPodiums:     2,
		NumTop5:        2,
		NumTop10:       2,
	})
	assertSkippedEvents(t, actual[0].SkipEventIDs)
	assertSkippedEvents(t, actual[1].SkipEventIDs)
}

func TestComputeCountsMultiRaceParticipationPerRace(t *testing.T) {
	compute := NewComputeStandings()

	actual := compute.Compute(&ComputeStandingsInput{
		EventIDs: []int32{40},
		Bookings: []*models.BookingEntry{
			bookingEntry(40, 401, 1, 5, "finish_pos"),
			bookingEntry(40, 401, 2, 7, "finish_pos"),
			bookingEntry(40, 402, 2, 5, "top_n_finishers"),
			bookingEntry(40, 402, 2, -2, "penalty_points"),
		},
		Participations: []Participation{
			testParticipation{raceGridID: 401, referenceID: 1, finishPosition: 1},
			testParticipation{raceGridID: 401, referenceID: 2, finishPosition: 2},
			testParticipation{raceGridID: 402, referenceID: 2, finishPosition: 3},
		},
		ReferenceID: func(booking *models.BookingEntry) int32 {
			return booking.DriverID.GetOrZero()
		},
	})

	if len(actual) != 2 {
		t.Fatalf("expected 2 standings entries, got %d", len(actual))
	}
	assertComputedStanding(t, actual[0], 2, &commonv1.StandingData{
		Position:       1,
		PrevPosition:   0,
		TotalPoints:    10,
		BonusPoints:    5,
		PenaltyPoints:  -2,
		NumEvents:      1,
		NumRaces:       2,
		NumPenaltyFree: 0,
		NumWins:        0,
		NumPodiums:     2,
		NumTop5:        2,
		NumTop10:       2,
	})
	assertComputedStanding(t, actual[1], 1, &commonv1.StandingData{
		Position:       2,
		PrevPosition:   0,
		TotalPoints:    5,
		BonusPoints:    0,
		PenaltyPoints:  0,
		NumEvents:      1,
		NumRaces:       1,
		NumPenaltyFree: 0,
		NumWins:        1,
		NumPodiums:     1,
		NumTop5:        1,
		NumTop10:       1,
	})
	assertSkippedEvents(t, actual[0].SkipEventIDs)
	assertSkippedEvents(t, actual[1].SkipEventIDs)
}

func TestComputeSkipModeAlwaysAppliesPerReference(t *testing.T) {
	compute := NewComputeStandings()

	actual := compute.Compute(&ComputeStandingsInput{
		EventIDs: []int32{10, 20, 30},
		Bookings: []*models.BookingEntry{
			bookingEntry(10, 100, 1, 10, "finish_pos"),
			bookingEntry(10, 100, 2, 8, "finish_pos"),
			bookingEntry(20, 200, 1, 2, "finish_pos"),
			bookingEntry(20, 200, 2, 6, "finish_pos"),
			bookingEntry(30, 300, 1, 7, "finish_pos"),
		},
		Participations: []Participation{
			testParticipation{raceGridID: 100, referenceID: 1, finishPosition: 1},
			testParticipation{raceGridID: 100, referenceID: 2, finishPosition: 2},
			testParticipation{raceGridID: 200, referenceID: 1, finishPosition: 4},
			testParticipation{raceGridID: 200, referenceID: 2, finishPosition: 1},
			testParticipation{raceGridID: 300, referenceID: 1, finishPosition: 2},
		},
		NumSkip:  1,
		SkipMode: SkipModeAlways,
		ReferenceID: func(booking *models.BookingEntry) int32 {
			return booking.DriverID.GetOrZero()
		},
	})

	if len(actual) != 2 {
		t.Fatalf("expected 2 standings entries, got %d", len(actual))
	}
	assertComputedStanding(t, actual[0], 1, &commonv1.StandingData{
		Position:       1,
		PrevPosition:   1,
		TotalPoints:    17,
		BonusPoints:    0,
		PenaltyPoints:  0,
		NumEvents:      3,
		NumRaces:       3,
		NumPenaltyFree: 0,
		NumWins:        1,
		NumPodiums:     2,
		NumTop5:        3,
		NumTop10:       3,
	})
	assertComputedStanding(t, actual[1], 2, &commonv1.StandingData{
		Position:       2,
		PrevPosition:   2,
		TotalPoints:    14,
		BonusPoints:    0,
		PenaltyPoints:  0,
		NumEvents:      2,
		NumRaces:       2,
		NumPenaltyFree: 0,
		NumWins:        1,
		NumPodiums:     2,
		NumTop5:        2,
		NumTop10:       2,
	})
	assertSkippedEvents(t, actual[0].SkipEventIDs, 20)
	assertSkippedEvents(t, actual[1].SkipEventIDs, 30)
}

func TestComputeSkipModeWhenApplicableRequiresThreshold(t *testing.T) {
	compute := NewComputeStandings()

	actual := compute.Compute(&ComputeStandingsInput{
		EventIDs: []int32{10, 20, 30},
		Bookings: []*models.BookingEntry{
			bookingEntry(10, 100, 1, 10, "finish_pos"),
			bookingEntry(20, 200, 1, 2, "finish_pos"),
			bookingEntry(30, 300, 1, 7, "finish_pos"),
		},
		Participations: []Participation{
			testParticipation{raceGridID: 100, referenceID: 1, finishPosition: 1},
			testParticipation{raceGridID: 200, referenceID: 1, finishPosition: 4},
			testParticipation{raceGridID: 300, referenceID: 1, finishPosition: 2},
		},
		NumTotalEvents: 5,
		NumSkip:        1,
		SkipMode:       SkipModeWhenApplicable,
		ReferenceID: func(booking *models.BookingEntry) int32 {
			return booking.DriverID.GetOrZero()
		},
	})

	if len(actual) != 1 {
		t.Fatalf("expected 1 standings entry, got %d", len(actual))
	}
	assertComputedStanding(t, actual[0], 1, &commonv1.StandingData{
		Position:       1,
		PrevPosition:   1,
		TotalPoints:    19,
		BonusPoints:    0,
		PenaltyPoints:  0,
		NumEvents:      3,
		NumRaces:       3,
		NumPenaltyFree: 0,
		NumWins:        1,
		NumPodiums:     2,
		NumTop5:        3,
		NumTop10:       3,
	})
	assertSkippedEvents(t, actual[0].SkipEventIDs)
}

func TestComputeBreaksTiesByFinishPositionsThenPenaltyThenBonus(t *testing.T) {
	compute := NewComputeStandings()

	actual := compute.Compute(&ComputeStandingsInput{
		EventIDs: []int32{10, 20},
		Bookings: []*models.BookingEntry{
			bookingEntry(10, 100, 1, 10, "finish_pos"),
			bookingEntry(10, 100, 2, 10, "finish_pos"),
			bookingEntry(20, 200, 1, 5, "finish_pos"),
			bookingEntry(20, 200, 2, 5, "finish_pos"),
		},
		Participations: []Participation{
			testParticipation{raceGridID: 100, referenceID: 1, finishPosition: 1},
			testParticipation{raceGridID: 100, referenceID: 2, finishPosition: 2},
			testParticipation{raceGridID: 200, referenceID: 1, finishPosition: 4},
			testParticipation{raceGridID: 200, referenceID: 2, finishPosition: 3},
		},
		ReferenceID: func(booking *models.BookingEntry) int32 {
			return booking.DriverID.GetOrZero()
		},
	})

	if len(actual) != 2 {
		t.Fatalf("expected 2 standings entries, got %d", len(actual))
	}
	assertComputedStanding(t, actual[0], 1, &commonv1.StandingData{
		Position:       1,
		PrevPosition:   1,
		TotalPoints:    15,
		BonusPoints:    0,
		PenaltyPoints:  0,
		NumEvents:      2,
		NumRaces:       2,
		NumPenaltyFree: 0,
		NumWins:        1,
		NumPodiums:     1,
		NumTop5:        2,
		NumTop10:       2,
	})
	assertComputedStanding(t, actual[1], 2, &commonv1.StandingData{
		Position:       2,
		PrevPosition:   2,
		TotalPoints:    15,
		BonusPoints:    0,
		PenaltyPoints:  0,
		NumEvents:      2,
		NumRaces:       2,
		NumPenaltyFree: 0,
		NumWins:        0,
		NumPodiums:     2,
		NumTop5:        2,
		NumTop10:       2,
	})

	actual = compute.Compute(&ComputeStandingsInput{
		EventIDs: []int32{30},
		Bookings: []*models.BookingEntry{
			bookingEntry(30, 300, 1, 10, "finish_pos"),
			bookingEntry(30, 300, 1, -2, "penalty_points"),
			bookingEntry(30, 300, 2, 10, "finish_pos"),
		},
		Participations: []Participation{
			testParticipation{raceGridID: 300, referenceID: 1, finishPosition: 2},
			testParticipation{raceGridID: 300, referenceID: 2, finishPosition: 2},
		},
		ReferenceID: func(booking *models.BookingEntry) int32 {
			return booking.DriverID.GetOrZero()
		},
	})

	assertComputedStanding(t, actual[0], 2, &commonv1.StandingData{
		Position:       1,
		PrevPosition:   0,
		TotalPoints:    10,
		BonusPoints:    0,
		PenaltyPoints:  0,
		NumEvents:      1,
		NumRaces:       1,
		NumPenaltyFree: 0,
		NumWins:        0,
		NumPodiums:     1,
		NumTop5:        1,
		NumTop10:       1,
	})
	assertComputedStanding(t, actual[1], 1, &commonv1.StandingData{
		Position:       2,
		PrevPosition:   0,
		TotalPoints:    8,
		BonusPoints:    0,
		PenaltyPoints:  -2,
		NumEvents:      1,
		NumRaces:       1,
		NumPenaltyFree: 0,
		NumWins:        0,
		NumPodiums:     1,
		NumTop5:        1,
		NumTop10:       1,
	})

	actual = compute.Compute(&ComputeStandingsInput{
		EventIDs: []int32{40},
		Bookings: []*models.BookingEntry{
			bookingEntry(40, 400, 1, 10, "finish_pos"),
			bookingEntry(40, 400, 1, 2, "fastest_lap"),
			bookingEntry(40, 400, 2, 10, "finish_pos"),
		},
		Participations: []Participation{
			testParticipation{raceGridID: 400, referenceID: 1, finishPosition: 2},
			testParticipation{raceGridID: 400, referenceID: 2, finishPosition: 2},
		},
		ReferenceID: func(booking *models.BookingEntry) int32 {
			return booking.DriverID.GetOrZero()
		},
	})

	assertComputedStanding(t, actual[0], 1, &commonv1.StandingData{
		Position:       1,
		PrevPosition:   0,
		TotalPoints:    12,
		BonusPoints:    2,
		PenaltyPoints:  0,
		NumEvents:      1,
		NumRaces:       1,
		NumPenaltyFree: 0,
		NumWins:        0,
		NumPodiums:     1,
		NumTop5:        1,
		NumTop10:       1,
	})
	assertComputedStanding(t, actual[1], 2, &commonv1.StandingData{
		Position:       2,
		PrevPosition:   0,
		TotalPoints:    10,
		BonusPoints:    0,
		PenaltyPoints:  0,
		NumEvents:      1,
		NumRaces:       1,
		NumPenaltyFree: 0,
		NumWins:        0,
		NumPodiums:     1,
		NumTop5:        1,
		NumTop10:       1,
	})
}

func assertComputedStanding(
	t *testing.T,
	standing *ComputedStanding,
	referenceID int32,
	expected *commonv1.StandingData,
) {
	t.Helper()

	if standing.ReferenceID != referenceID {
		t.Fatalf(
			"unexpected reference ID: got %d want %d",
			standing.ReferenceID,
			referenceID,
		)
	}

	assertStanding(t, standing.StandingData, expected)
}

func bookingEntry(
	eventID int32,
	raceGridID int32,
	driverID int32,
	points int32,
	sourceType string,
) *models.BookingEntry {
	return &models.BookingEntry{
		EventID:    eventID,
		RaceGridID: null.From(raceGridID),
		DriverID:   null.From(driverID),
		SourceType: mytypes.SourceType(sourceType),
		Points:     points,
	}
}

func assertStanding(
	t *testing.T,
	standing *commonv1.StandingData,
	expected *commonv1.StandingData,
) {
	t.Helper()

	if standing.Position != expected.Position {
		t.Fatalf(
			"unexpected position: got %d want %d",
			standing.Position,
			expected.Position,
		)
	}
	if standing.PrevPosition != expected.PrevPosition {
		t.Fatalf(
			"unexpected prev position: got %d want %d",
			standing.PrevPosition,
			expected.PrevPosition,
		)
	}
	if standing.TotalPoints != expected.TotalPoints {
		t.Fatalf(
			"unexpected total points: got %d want %d",
			standing.TotalPoints,
			expected.TotalPoints,
		)
	}
	if standing.BonusPoints != expected.BonusPoints {
		t.Fatalf(
			"unexpected bonus points: got %d want %d",
			standing.BonusPoints,
			expected.BonusPoints,
		)
	}
	if standing.PenaltyPoints != expected.PenaltyPoints {
		t.Fatalf(
			"unexpected penalty points: got %d want %d",
			standing.PenaltyPoints,
			expected.PenaltyPoints,
		)
	}
	if standing.NumEvents != expected.NumEvents {
		t.Fatalf(
			"unexpected num events: got %d want %d",
			standing.NumEvents,
			expected.NumEvents,
		)
	}
	if standing.NumRaces != expected.NumRaces {
		t.Fatalf(
			"unexpected num races: got %d want %d",
			standing.NumRaces,
			expected.NumRaces,
		)
	}
	if standing.NumPenaltyFree != expected.NumPenaltyFree {
		t.Fatalf(
			"unexpected num penalty free: got %d want %d",
			standing.NumPenaltyFree,
			expected.NumPenaltyFree,
		)
	}
	if standing.NumWins != expected.NumWins {
		t.Fatalf(
			"unexpected num wins: got %d want %d",
			standing.NumWins,
			expected.NumWins,
		)
	}
	if standing.NumPodiums != expected.NumPodiums {
		t.Fatalf(
			"unexpected num podiums: got %d want %d",
			standing.NumPodiums,
			expected.NumPodiums,
		)
	}
	if standing.NumTop5 != expected.NumTop5 {
		t.Fatalf(
			"unexpected num top5: got %d want %d",
			standing.NumTop5,
			expected.NumTop5,
		)
	}
	if standing.NumTop10 != expected.NumTop10 {
		t.Fatalf(
			"unexpected num top10: got %d want %d",
			standing.NumTop10,
			expected.NumTop10,
		)
	}
}

func assertSkippedEvents(
	t *testing.T,
	actual []int32,
	expected ...int32,
) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf(
			"unexpected skipped event count: got %d want %d",
			len(actual),
			len(expected),
		)
	}

	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf(
				"unexpected skipped event at %d: got %d want %d",
				index,
				actual[index],
				expected[index],
			)
		}
	}
}
