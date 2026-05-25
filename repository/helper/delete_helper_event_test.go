//nolint:lll,funlen,dupl // test code can be verbose
package helper

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/lib/pq"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	bobtypes "github.com/stephenafamo/bob/types"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/testhelpers"
)

type eventFixture struct {
	event          *models.Event
	race           *models.Race
	raceGrid       *models.RaceGrid
	resultEntry    *models.ResultEntry
	bookingEntry   *models.BookingEntry
	importBatch    *models.ImportBatch
	driverStanding *models.EventDriverStanding
	teamStanding   *models.EventTeamStanding
	auditEntry     *models.EventProcessingAudit
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
	assertMissingEventDriverStanding(t, ctx, target.driverStanding.ID)
	assertMissingEventTeamStanding(t, ctx, target.teamStanding.ID)
	assertMissingEventProcessingAudit(t, ctx, target.auditEntry.ID)
	assertMissingRace(t, ctx, target.race.ID)
	assertEventExists(t, ctx, target.event.ID)

	assertRaceGridExists(t, ctx, other.raceGrid.ID)
	assertResultEntryExists(t, ctx, other.resultEntry.ID)
	assertBookingEntryExists(t, ctx, other.bookingEntry.ID)
	assertImportBatchExists(t, ctx, other.importBatch.ID)
	assertEventDriverStandingExists(t, ctx, other.driverStanding.ID)
	assertEventTeamStandingExists(t, ctx, other.teamStanding.ID)
	assertEventProcessingAuditExists(t, ctx, other.auditEntry.ID)
	assertRaceExists(t, ctx, other.race.ID)
	assertEventExists(t, ctx, other.event.ID)
}

func TestDeleteRaceRelated(t *testing.T) {
	ctx := newTxContext(t)

	target := seedEventFixture(t, ctx, "target")
	other := seedEventFixture(t, ctx, "other")

	if err := DeleteRaceRelated(ctx, target.race.ID); err != nil {
		t.Fatalf("DeleteRaceRelated returned error: %v", err)
	}

	assertMissingRaceGrid(t, ctx, target.raceGrid.ID)
	assertMissingResultEntry(t, ctx, target.resultEntry.ID)
	assertMissingBookingEntry(t, ctx, target.bookingEntry.ID)
	assertMissingImportBatch(t, ctx, target.importBatch.ID)
	assertEventExists(t, ctx, target.event.ID)

	assertRaceExists(t, ctx, other.race.ID)
	assertRaceGridExists(t, ctx, other.raceGrid.ID)
	assertResultEntryExists(t, ctx, other.resultEntry.ID)
	assertBookingEntryExists(t, ctx, other.bookingEntry.ID)
	assertImportBatchExists(t, ctx, other.importBatch.ID)
}

func TestDeleteRaceGridRelated(t *testing.T) {
	ctx := newTxContext(t)

	target := seedEventFixture(t, ctx, "target")
	other := seedEventFixture(t, ctx, "other")

	if err := DeleteRaceGridRelated(ctx, target.raceGrid.ID); err != nil {
		t.Fatalf("DeleteRaceGridRelated returned error: %v", err)
	}

	assertRaceExists(t, ctx, target.race.ID)
	assertRaceGridExists(t, ctx, target.raceGrid.ID)
	assertMissingResultEntry(t, ctx, target.resultEntry.ID)
	assertMissingBookingEntry(t, ctx, target.bookingEntry.ID)
	assertMissingImportBatch(t, ctx, target.importBatch.ID)

	assertRaceExists(t, ctx, other.race.ID)
	assertRaceGridExists(t, ctx, other.raceGrid.ID)
	assertResultEntryExists(t, ctx, other.resultEntry.ID)
	assertBookingEntryExists(t, ctx, other.bookingEntry.ID)
	assertImportBatchExists(t, ctx, other.importBatch.ID)
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
		{name: "DeleteEventDriverStandings", fn: DeleteEventDriverStandings},
		{name: "DeleteEventTeamStandings", fn: DeleteEventTeamStandings},
		{name: "DeleteEventProcessingAudit", fn: DeleteEventProcessingAuditForEvent},
		{name: "DeleteRaceRelated", fn: DeleteRaceRelated},
		{name: "DeleteRaceBookingEntries", fn: DeleteRaceBookingEntries},
		{name: "DeleteRaceResultEntries", fn: DeleteRaceResultEntries},
		{name: "DeleteRaceImportBatches", fn: DeleteRaceImportBatches},
		{name: "DeleteRaceRaceGrids", fn: DeleteRaceRaceGrids},
		{name: "DeleteRaceGridRelated", fn: DeleteRaceGridRelated},
		{name: "DeleteRaceGridBookingEntries", fn: DeleteRaceGridBookingEntries},
		{name: "DeleteRaceGridResultEntries", fn: DeleteRaceGridResultEntries},
		{name: "DeleteRaceGridImportBatches", fn: DeleteRaceGridImportBatches},
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
	seedStandingsAndAuditDependents(t, ctx, suffix, &base)

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

func assertEventDriverStandingExists(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertExists(
		t,
		ctx,
		"event driver standing",
		queryEventDriverStandingByID(ctx, getExecutorFromContext(t, ctx), id),
	)
}

func assertMissingEventDriverStanding(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertMissing(
		t,
		ctx,
		"event driver standing",
		queryEventDriverStandingByID(ctx, getExecutorFromContext(t, ctx), id),
	)
}

func assertEventTeamStandingExists(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertExists(
		t,
		ctx,
		"event team standing",
		queryEventTeamStandingByID(ctx, getExecutorFromContext(t, ctx), id),
	)
}

func assertMissingEventTeamStanding(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertMissing(
		t,
		ctx,
		"event team standing",
		queryEventTeamStandingByID(ctx, getExecutorFromContext(t, ctx), id),
	)
}

func assertEventProcessingAuditExists(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertExists(
		t,
		ctx,
		"event processing audit",
		queryEventProcessingAuditByID(ctx, getExecutorFromContext(t, ctx), id),
	)
}

func assertMissingEventProcessingAudit(t *testing.T, ctx context.Context, id int32) {
	t.Helper()
	assertMissing(
		t,
		ctx,
		"event processing audit",
		queryEventProcessingAuditByID(ctx, getExecutorFromContext(t, ctx), id),
	)
}

func queryEventDriverStandingByID(ctx context.Context, exec bob.Executor, id int32) error {
	_, err := models.EventDriverStandings.Query(
		sm.Where(models.EventDriverStandings.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, exec)
	return err
}

func queryEventTeamStandingByID(ctx context.Context, exec bob.Executor, id int32) error {
	_, err := models.EventTeamStandings.Query(
		sm.Where(models.EventTeamStandings.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, exec)
	return err
}

func queryEventProcessingAuditByID(ctx context.Context, exec bob.Executor, id int32) error {
	_, err := models.EventProcessingAudits.Query(
		sm.Where(models.EventProcessingAudits.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, exec)
	return err
}

//nolint:whitespace //editor/linter issue
func seedStandingsAndAuditDependents(
	t *testing.T,
	ctx context.Context,
	suffix string,
	fixture *eventFixture,
) {
	t.Helper()

	exec := getExecutorFromContext(t, ctx)

	driver, err := models.Drivers.Insert(&models.DriverSetter{
		ExternalID: omit.From("driver-" + suffix),
		Name:       omit.From("Driver " + suffix),
		IsActive:   omit.From(true),
		CreatedBy:  omit.From(testhelpers.TestUserSeed),
		UpdatedBy:  omit.From(testhelpers.TestUserSeed),
	}).One(ctx, exec)
	if err != nil {
		t.Fatalf("failed to seed driver standing dependency: %v", err)
	}

	team, err := models.Teams.Insert(&models.TeamSetter{
		SeasonID:  omit.From(fixture.event.SeasonID),
		Name:      omit.From("Team " + suffix),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(testhelpers.TestUserSeed),
		UpdatedBy: omit.From(testhelpers.TestUserSeed),
	}).One(ctx, exec)
	if err != nil {
		t.Fatalf("failed to seed team standing dependency: %v", err)
	}

	fixture.driverStanding, err = models.EventDriverStandings.Insert(&models.EventDriverStandingSetter{
		EventID:         omit.From(fixture.event.ID),
		SeasonID:        omit.From(fixture.event.SeasonID),
		DriverID:        omit.From(driver.ID),
		Position:        omit.From(int32(1)),
		TotalPoints:     omit.From(int32(25)),
		DroppedEventIds: omit.From(pq.Int32Array{}),
		CreatedBy:       omit.From(testhelpers.TestUserSeed),
		UpdatedBy:       omit.From(testhelpers.TestUserSeed),
	}).
		One(ctx, exec)
	if err != nil {
		t.Fatalf("failed to seed event driver standing: %v", err)
	}

	fixture.teamStanding, err = models.EventTeamStandings.Insert(&models.EventTeamStandingSetter{
		EventID:         omit.From(fixture.event.ID),
		SeasonID:        omit.From(fixture.event.SeasonID),
		TeamID:          omit.From(team.ID),
		Position:        omit.From(int32(1)),
		TotalPoints:     omit.From(int32(25)),
		DroppedEventIds: omit.From(pq.Int32Array{}),
		CreatedBy:       omit.From(testhelpers.TestUserSeed),
		UpdatedBy:       omit.From(testhelpers.TestUserSeed),
	}).One(ctx, exec)
	if err != nil {
		t.Fatalf("failed to seed event team standing: %v", err)
	}

	fixture.auditEntry, err = models.EventProcessingAudits.Insert(&models.EventProcessingAuditSetter{
		EventID:     omit.From(fixture.event.ID),
		ToState:     omit.From("raw_imported"),
		Action:      omit.From("seeded"),
		PayloadJSON: omit.From(bobtypes.NewJSON[json.RawMessage]([]byte("{}"))),
		CreatedBy:   omit.From(testhelpers.TestUserSeed),
		UpdatedBy:   omit.From(testhelpers.TestUserSeed),
	}).
		One(ctx, exec)
	if err != nil {
		t.Fatalf("failed to seed event processing audit: %v", err)
	}
}

func getExecutorFromContext(t *testing.T, ctx context.Context) bob.Executor {
	t.Helper()

	exec := pgbob.FromContext(ctx)
	if exec == nil {
		t.Fatal("expected bob executor in context")
	}

	return exec
}
