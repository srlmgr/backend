package impl

import (
	"testing"

	"github.com/aarondl/opt/null"

	"github.com/srlmgr/backend/db/models"
)

func TestRecomputeFinishPosByClassAndGridIgnoresExcludedEntries(t *testing.T) {
	t.Parallel()

	normalFirst := &models.ResultEntry{
		CarClassID:     null.From(int32(10)),
		RaceGridID:     20,
		FinishPosition: 1,
	}
	ignored := &models.ResultEntry{
		CarClassID:     null.From(int32(10)),
		RaceGridID:     20,
		FinishPosition: 2,
		State:          models.ResultEntryStateIgnore,
	}
	normalSecond := &models.ResultEntry{
		CarClassID:     null.From(int32(10)),
		RaceGridID:     20,
		FinishPosition: 3,
	}

	results := (&standingProc{
		resultEntries: []*models.ResultEntry{normalFirst, ignored, normalSecond},
	}).recomputeFinishPosByClassAndGrid()

	classResults := results[10]
	if len(classResults) != 2 {
		t.Fatalf("unexpected result count: got %d want 2", len(classResults))
	}
	if normalFirst.FinishPosition != 1 || normalSecond.FinishPosition != 2 {
		t.Fatalf(
			"unexpected recomputed positions: got %d and %d",
			normalFirst.FinishPosition,
			normalSecond.FinishPosition,
		)
	}
}
