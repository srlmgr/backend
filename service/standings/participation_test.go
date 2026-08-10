package standings

import (
	"testing"

	"github.com/aarondl/opt/null"

	"github.com/srlmgr/backend/db/models"
)

func TestParticipationsFromResultEntries(t *testing.T) {
	actual := ParticipationsFromResultEntries([]*models.ResultEntry{
		nil,
		{
			RaceGridID:     10,
			DriverID:       null.From(int32(5)),
			FinishPosition: 2,
		},
		{
			RaceGridID:     20,
			DriverID:       null.From(int32(7)),
			FinishPosition: 1,
		},
	}, func(entry *models.ResultEntry) int32 {
		return entry.DriverID.GetOrZero()
	})

	if len(actual) != 2 {
		t.Fatalf("expected 2 participations, got %d", len(actual))
	}

	assertParticipation(t, actual[0], 10, 5, 2)
	assertParticipation(t, actual[1], 20, 7, 1)
}

//nolint:whitespace // editor/linter issue
func assertParticipation(
	t *testing.T,
	participation Participation,
	raceGridID int32,
	referenceID int32,
	finishPosition int32,
) {
	t.Helper()

	if participation.RaceGridID() != raceGridID {
		t.Fatalf(
			"unexpected race grid ID: got %d want %d",
			participation.RaceGridID(),
			raceGridID,
		)
	}
	if participation.ReferenceID() != referenceID {
		t.Fatalf(
			"unexpected reference ID: got %d want %d",
			participation.ReferenceID(),
			referenceID,
		)
	}
	if participation.FinishPosition() != finishPosition {
		t.Fatalf(
			"unexpected finish position: got %d want %d",
			participation.FinishPosition(),
			finishPosition,
		)
	}
}
