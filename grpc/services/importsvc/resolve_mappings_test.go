package importsvc

import (
	"testing"
	"time"

	"github.com/aarondl/opt/null"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/grpc/services/importsvc/importer"
)

//nolint:whitespace // editor/linter issue
func TestBuildResolvedResultEntrySetter(t *testing.T) {
	entry := &models.ResultEntry{
		State:             "mapping_error",
		TeamID:            null.From(int32(12)),
		TeamDrivers:       null.From(mytypes.TeamDrivers{DriverIDs: []int32{3, 4}}),
		CarModelVariantID: null.From(int32(7)),
		CarClassID:        null.From(int32(8)),
	}
	now := time.Now()
	setter := buildResolvedResultEntrySetter(entry, now, "user")
	if !setter.DriverID.IsNull() || setter.TeamID.MustGet() != 12 ||
		setter.CarModelVariantID.MustGet() != 7 || setter.CarClassID.MustGet() != 8 {
		t.Fatalf("unexpected IDs: %+v", setter)
	}
	drivers := setter.TeamDrivers.MustGet().DriverIDs
	if len(drivers) != 2 || drivers[0] != 3 || drivers[1] != 4 ||
		setter.State.MustGet() != entry.State || setter.UpdatedBy.MustGet() != "user" ||
		!setter.UpdatedAt.MustGet().Equal(now) {
		t.Fatalf("unexpected persisted fields: %+v", setter)
	}
	entry.TeamID = null.Val[int32]{}
	entry.TeamDrivers = null.Val[mytypes.TeamDrivers]{}
	entry.CarModelVariantID = null.Val[int32]{}
	entry.CarClassID = null.Val[int32]{}
	setter = buildResolvedResultEntrySetter(entry, now, "user")
	if !setter.TeamID.IsNull() || !setter.TeamDrivers.IsNull() ||
		!setter.CarModelVariantID.IsNull() || !setter.CarClassID.IsNull() {
		t.Fatalf("expected cleared IDs and team drivers: %+v", setter)
	}
}

//nolint:whitespace // editor/linter issue
func TestParsedInputWithResultEntriesUsesEditedValues(t *testing.T) {
	input := &importer.ParsedImportPayload{
		Session: importer.SessionInfo{Track: "track"},
		Results: []*importer.ResultRow{{
			DriverID:  "source-driver",
			Name:      "Original Driver",
			CarID:     "source-car",
			Car:       "Original Car",
			CarNumber: "12",
		}},
	}
	entries := []*models.ResultEntry{{
		FinishPosition: 2,
		RawDriverName:  null.From("Edited Driver"),
		RawCarName:     null.From("Edited Car"),
		CarNumber:      null.From("27"),
		StartPosition:  null.From(int32(3)),
		LapsCompleted:  40,
		Incidents:      null.From(int32(1)),
		Offtracks:      2,
	}}

	got, err := parsedInputWithResultEntries(input, entries, false)
	if err != nil {
		t.Fatal(err)
	}
	row := got.Results[0]
	if got.Session.Track != "track" || row.FinPos != 2 || row.Name != "Edited Driver" ||
		row.DriverID != "" || row.Car != "Edited Car" || row.CarID != "" ||
		row.CarNumber != "27" || row.StartPos != 3 || row.Laps != 40 ||
		row.Incidents != 1 || row.Offtracks != 2 {
		t.Fatalf("unexpected rebuilt input: %+v", got)
	}
	if input.Results[0].Name != "Original Driver" ||
		input.Results[0].Car != "Original Car" {
		t.Fatal("rebuilding input mutated the original parsed payload")
	}
}

//nolint:whitespace // editor/linter issue
func TestParsedInputMatchesEntriesByCreationOrder(t *testing.T) {
	input := &importer.ParsedImportPayload{
		Results: []*importer.ResultRow{
			{DriverID: "driver-a", Name: "Driver A"},
			{DriverID: "driver-b", Name: "Driver B"},
		},
	}
	entries := []*models.ResultEntry{
		{ID: 2, FinishPosition: 1, RawDriverName: null.From("Edited B")},
		{ID: 1, FinishPosition: 2, RawDriverName: null.From("Driver A")},
	}

	got, err := parsedInputWithResultEntries(input, entries, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Results[0].Name != "Edited B" || got.Results[0].DriverID != "" ||
		got.Results[1].Name != "Driver A" || got.Results[1].DriverID != "driver-a" {
		t.Fatalf("source rows were paired with the wrong result entries: %+v", got.Results)
	}
}

//nolint:whitespace // editor/linter issue
func TestMergeResolvedEntriesPreservesExistingFields(t *testing.T) {
	existing := []*models.ResultEntry{{
		ID:                12,
		RaceGridID:        5,
		IsGuestStarter:    true,
		RawDriverName:     null.From("Driver"),
		RawCarName:        null.From("Car"),
		AdminNotes:        null.From("keep"),
		DriverID:          null.From(int32(3)),
		TeamID:            null.From(int32(4)),
		TeamDrivers:       null.From(mytypes.TeamDrivers{DriverIDs: []int32{3, 4}}),
		CarModelVariantID: null.From(int32(5)),
		CarClassID:        null.From(int32(6)),
		State:             "mapping_error",
	}}
	result := &importer.Result{Entries: []*models.ResultEntry{{
		State:             "normal",
		DriverID:          null.From(int32(30)),
		TeamID:            null.From(int32(40)),
		TeamDrivers:       null.From(mytypes.TeamDrivers{DriverIDs: []int32{30, 40}}),
		CarModelVariantID: null.From(int32(50)),
		CarClassID:        null.From(int32(60)),
	}}}

	if err := mergeResolvedEntries(result, existing); err != nil {
		t.Fatal(err)
	}
	got := result.Entries[0]
	if got.ID != 12 || got.RaceGridID != 5 || !got.IsGuestStarter ||
		got.AdminNotes.GetOr("") != "keep" || got.State != "normal" ||
		got.DriverID.GetOrZero() != 30 || got.TeamID.GetOrZero() != 40 ||
		got.TeamDrivers.GetOrZero().DriverIDs[0] != 30 ||
		got.CarModelVariantID.GetOrZero() != 50 || got.CarClassID.GetOrZero() != 60 {
		t.Fatalf("unexpected merged entry: %+v", got)
	}
	if existing[0].DriverID.GetOrZero() != 3 ||
		existing[0].State != "mapping_error" {
		t.Fatal("merging resolved values mutated the existing entry")
	}
}
