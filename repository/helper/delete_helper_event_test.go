//nolint:lll // test code can be verbose
package helper

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/testhelpers"
)

type eventFixture struct {
	event        *models.Event
	race         *models.Race
	raceGrid     *models.RaceGrid
	resultEntry  *models.ResultEntry
	bookingEntry *models.BookingEntry
	importBatch  *models.ImportBatch
}

func TestDeleteEventRelated(t *testing.T) {
	ctx := newTxContext(t)

	target := seedEventFixture(t, ctx, "target")
	other := seedEventFixture(t, ctx, "other")

	if err := DeleteEventRelated(ctx, target.event.ID); err != nil {
		t.Fatalf("DeleteEventRelated returned error: %v", err)
	}

	assertMissingRaceGrid(t, ctx, target.raceGrid.ID)
	assertMissingResultEntry(t, ctx, target.resultEntry.ID)
	assertMissingBookingEntry(t, ctx, target.bookingEntry.ID)
	assertMissingImportBatch(t, ctx, target.importBatch.ID)
	assertMissingRace(t, ctx, target.race.ID)
	assertEventExists(t, ctx, target.event.ID)

	assertRaceGridExists(t, ctx, other.raceGrid.ID)
	assertResultEntryExists(t, ctx, other.resultEntry.ID)
	assertBookingEntryExists(t, ctx, other.bookingEntry.ID)
	assertImportBatchExists(t, ctx, other.importBatch.ID)
	assertRaceExists(t, ctx, other.race.ID)
	assertEventExists(t, ctx, other.event.ID)
}

func TestDeleteEventRaceGrids(t *testing.T) {
	ctx := newTxContext(t)

	target := seedEventFixture(t, ctx, "target")
	other := seedEventFixture(t, ctx, "other")

	deleteEventDependentsExceptRaceGrid(t, ctx, target)

	if err := DeleteEventRaceGrids(ctx, target.event.ID); err != nil {
		t.Fatalf("DeleteEventRaceGrids returned error: %v", err)
	}

	assertMissingRaceGrid(t, ctx, target.raceGrid.ID)
	assertRaceExists(t, ctx, target.race.ID)
	assertRaceGridExists(t, ctx, other.raceGrid.ID)
	assertRaceExists(t, ctx, other.race.ID)
}

func TestDeleteEventBookingEntries(t *testing.T) {
	ctx := newTxContext(t)

	target := seedEventFixture(t, ctx, "target")
	other := seedEventFixture(t, ctx, "other")

	if err := DeleteEventBookingEntries(ctx, target.event.ID); err != nil {
		t.Fatalf("DeleteEventBookingEntries returned error: %v", err)
	}

	assertMissingBookingEntry(t, ctx, target.bookingEntry.ID)
	assertResultEntryExists(t, ctx, target.resultEntry.ID)
	assertBookingEntryExists(t, ctx, other.bookingEntry.ID)
	assertResultEntryExists(t, ctx, other.resultEntry.ID)
}

func TestDeleteEventRaces(t *testing.T) {
	ctx := newTxContext(t)

	target := seedEventBase(t, ctx, "target")
	other := seedEventBase(t, ctx, "other")
	targetRace := testhelpers.SeedRaceContext(
		t,
		ctx,
		target.event.ID,
		"Race target",
		"race",
		1,
	)
	otherRace := testhelpers.SeedRaceContext(
		t,
		ctx,
		other.event.ID,
		"Race other",
		"race",
		1,
	)

	if err := DeleteEventRaces(ctx, target.event.ID); err != nil {
		t.Fatalf("DeleteEventRaces returned error: %v", err)
	}

	assertMissingRace(t, ctx, targetRace.ID)
	assertRaceExists(t, ctx, otherRace.ID)
	assertEventExists(t, ctx, target.event.ID)
	assertEventExists(t, ctx, other.event.ID)
}

func TestDeleteEventResultEntries(t *testing.T) {
	ctx := newTxContext(t)

	target := seedEventFixture(t, ctx, "target")
	other := seedEventFixture(t, ctx, "other")

	deleteBookingEntry(t, ctx, target.bookingEntry.ID)
	deleteBookingEntry(t, ctx, other.bookingEntry.ID)

	if err := DeleteEventResultEntries(ctx, target.event.ID); err != nil {
		t.Fatalf("DeleteEventResultEntries returned error: %v", err)
	}

	assertMissingResultEntry(t, ctx, target.resultEntry.ID)
	assertRaceGridExists(t, ctx, target.raceGrid.ID)
	assertResultEntryExists(t, ctx, other.resultEntry.ID)
	assertRaceGridExists(t, ctx, other.raceGrid.ID)
}

func TestDeleteEventImportBatches(t *testing.T) {
	ctx := newTxContext(t)

	target := seedEventFixture(t, ctx, "target")
	other := seedEventFixture(t, ctx, "other")

	if err := DeleteEventImportBatches(ctx, target.event.ID); err != nil {
		t.Fatalf("DeleteEventImportBatches returned error: %v", err)
	}

	assertMissingImportBatch(t, ctx, target.importBatch.ID)
	assertRaceExists(t, ctx, target.race.ID)
	assertImportBatchExists(t, ctx, other.importBatch.ID)
	assertRaceExists(t, ctx, other.race.ID)
}

func TestDeleteEventHelpersRequireExecutor(t *testing.T) {
	tests := []struct {
		name string
		fn   func(context.Context, int32) error
	}{
		{name: "DeleteEventRelated", fn: DeleteEventRelated},
		{name: "DeleteEventRaceGrids", fn: DeleteEventRaceGrids},
		{name: "DeleteEventBookingEntries", fn: DeleteEventBookingEntries},
		{name: "DeleteEventRaces", fn: DeleteEventRaces},
		{name: "DeleteEventResultEntries", fn: DeleteEventResultEntries},
		{name: "DeleteEventImportBatches", fn: DeleteEventImportBatches},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(context.Background(), 123)
			if !errors.Is(err, ErrNoBobExecutorInContext) {
				t.Fatalf("expected ErrNoBobExecutorInContext, got %v", err)
			}
		})
	}
}

func seedEventFixture(t *testing.T, ctx context.Context, suffix string) eventFixture {
	t.Helper()

	base := seedEventBase(t, ctx, suffix)
	base.race = testhelpers.SeedRaceContext(
		t,
		ctx,
		base.event.ID,
		"Race "+suffix,
		"race",
		1,
	)
	base.raceGrid = testhelpers.SeedRaceGridContext(
		t,
		ctx,
		base.race.ID,
		"Grid "+suffix,
		"race",
		1,
	)
	base.resultEntry = testhelpers.SeedResultEntryContext(
		t,
		ctx,
		base.raceGrid.ID,
		"Driver "+suffix,
		1,
	)
	base.bookingEntry = testhelpers.SeedBookingEntryContext(
		t,
		ctx,
		base.event.ID,
		base.race.ID,
		base.raceGrid.ID,
		"Booking "+suffix,
		10,
	)
	base.importBatch = testhelpers.SeedImportBatchContext(
		t,
		ctx,
		base.raceGrid.ID,
		suffix+".json",
	)

	return base
}

func seedEventBase(t *testing.T, ctx context.Context, suffix string) eventFixture {
	t.Helper()

	sim := testhelpers.SeedRacingSimContext(t, ctx, "Sim "+suffix)
	series := testhelpers.SeedSeriesContext(t, ctx, sim.ID, "Series "+suffix)
	pointSystem := testhelpers.SeedPointSystemContext(t, ctx, "Point System "+suffix)
	season := testhelpers.SeedSeasonContext(
		t,
		ctx,
		series.ID,
		pointSystem.ID,
		"Season "+suffix,
	)
	track := testhelpers.SeedTrackContext(t, ctx, "Track "+suffix)
	trackLayout := testhelpers.SeedTrackLayoutContext(t, ctx, track.ID, "Layout "+suffix)

	return eventFixture{
		event: testhelpers.SeedEventContext(
			t,
			ctx,
			season.ID,
			trackLayout.ID,
			"Event "+suffix,
			1,
		),
	}
}

//nolint:whitespace // editor/linter issue
func deleteEventDependentsExceptRaceGrid(
	t *testing.T,
	ctx context.Context,
	fixture eventFixture,
) {
	t.Helper()

	deleteBookingEntry(t, ctx, fixture.bookingEntry.ID)
	deleteResultEntry(t, ctx, fixture.resultEntry.ID)
	deleteImportBatch(t, ctx, fixture.importBatch.ID)
}

func deleteBookingEntry(t *testing.T, ctx context.Context, id int32) {
	t.Helper()

	_, err := models.BookingEntries.Delete(
		dm.Where(models.BookingEntries.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to delete booking entry %d during test setup: %v", id, err)
	}
}

func deleteResultEntry(t *testing.T, ctx context.Context, id int32) {
	t.Helper()

	_, err := models.ResultEntries.Delete(
		dm.Where(models.ResultEntries.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to delete result entry %d during test setup: %v", id, err)
	}
}

func deleteImportBatch(t *testing.T, ctx context.Context, id int32) {
	t.Helper()

	_, err := models.ImportBatches.Delete(
		dm.Where(models.ImportBatches.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, getExecutorFromContext(t, ctx))
	if err != nil {
		t.Fatalf("failed to delete import batch %d during test setup: %v", id, err)
	}
}

func assertEventExists(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertExists(t, ctx, "event", queryEventByID(ctx, getExecutorFromContext(t, ctx), id))
}

func assertRaceExists(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertExists(t, ctx, "race", queryRaceByID(ctx, getExecutorFromContext(t, ctx), id))
}

func assertMissingRace(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertMissing(t, ctx, "race", queryRaceByID(ctx, getExecutorFromContext(t, ctx), id))
}

func assertRaceGridExists(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	exec := getExecutorFromContext(t, ctx)
	assertExists(t, ctx, "race grid", queryRaceGridByID(ctx, exec, id))
}

func assertMissingRaceGrid(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	exec := getExecutorFromContext(t, ctx)
	assertMissing(t, ctx, "race grid", queryRaceGridByID(ctx, exec, id))
}

func assertResultEntryExists(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	exec := getExecutorFromContext(t, ctx)
	assertExists(t, ctx, "result entry", queryResultEntryByID(ctx, exec, id))
}

func assertMissingResultEntry(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	exec := getExecutorFromContext(t, ctx)
	assertMissing(t, ctx, "result entry", queryResultEntryByID(ctx, exec, id))
}

func assertBookingEntryExists(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	exec := getExecutorFromContext(t, ctx)
	assertExists(t, ctx, "booking entry", queryBookingEntryByID(ctx, exec, id))
}

func assertMissingBookingEntry(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertMissing(
		t,
		ctx,
		"booking entry",
		queryBookingEntryByID(ctx, getExecutorFromContext(t, ctx), id),
	)
}

func assertImportBatchExists(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertExists(
		t,
		ctx,
		"import batch",
		queryImportBatchByID(ctx, getExecutorFromContext(t, ctx), id),
	)
}

func assertMissingImportBatch(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertMissing(
		t,
		ctx,
		"import batch",
		queryImportBatchByID(ctx, getExecutorFromContext(t, ctx), id),
	)
}

func assertExists(t *testing.T, _ context.Context, label string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("expected %s to exist: %v", label, err)
	}
}

func assertMissing(t *testing.T, _ context.Context, label string, err error) {
	t.Helper()

	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		t.Fatalf("expected %s lookup to fail with sql.ErrNoRows, got %v", label, err)
	}
	t.Fatalf("expected %s to be deleted", label)
}

func queryEventByID(ctx context.Context, exec bob.Executor, id int32) error {
	_, err := models.Events.Query(sm.Where(models.Events.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, exec)
	return err
}

func queryRaceByID(ctx context.Context, exec bob.Executor, id int32) error {
	_, err := models.Races.Query(
		sm.Where(models.Races.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, exec)
	return err
}

func queryRaceGridByID(ctx context.Context, exec bob.Executor, id int32) error {
	_, err := models.RaceGrids.Query(sm.Where(models.RaceGrids.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, exec)
	return err
}

func queryResultEntryByID(ctx context.Context, exec bob.Executor, id int32) error {
	_, err := models.ResultEntries.Query(sm.Where(models.ResultEntries.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, exec)
	return err
}

func queryBookingEntryByID(ctx context.Context, exec bob.Executor, id int32) error {
	_, err := models.BookingEntries.Query(sm.Where(models.BookingEntries.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, exec)
	return err
}

func queryImportBatchByID(ctx context.Context, exec bob.Executor, id int32) error {
	_, err := models.ImportBatches.Query(sm.Where(models.ImportBatches.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, exec)
	return err
}

func getExecutorFromContext(t *testing.T, ctx context.Context) bob.Executor {
	t.Helper()

	exec := pgbob.FromContext(ctx)
	if exec == nil {
		t.Fatal("expected bob executor in context")
	}

	return exec
}
