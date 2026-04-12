//nolint:lll,dupl,funlen // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/srlmgr/backend/authn"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/services/conversion"
)

//nolint:gocyclo // much to do
func TestResultEntrySetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (resultEntrySetterBuilder{}).Build(&v1.CreateResultEntryRequest{
		RaceGridId:        10,
		DriverId:          5,
		CarModelId:        3,
		FinishingPosition: 1,
		CompletedLaps:     25,
		FastestLapTimeMs:  90000,
		Incidents:         2,
		State:             commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL,
		AdminNotes:        "test note",
	})

	if !setter.RaceGridID.IsValue() || setter.RaceGridID.MustGet() != 10 {
		t.Fatalf("unexpected race_grid_id setter value: %+v", setter.RaceGridID)
	}
	if !setter.DriverID.IsValue() || setter.DriverID.MustGet() != 5 {
		t.Fatalf("unexpected driver_id setter value: %+v", setter.DriverID)
	}
	if !setter.CarModelID.IsValue() || setter.CarModelID.MustGet() != 3 {
		t.Fatalf("unexpected car_model_id setter value: %+v", setter.CarModelID)
	}
	if !setter.FinishPosition.IsValue() || setter.FinishPosition.MustGet() != 1 {
		t.Fatalf("unexpected finish_position setter value: %+v", setter.FinishPosition)
	}
	if !setter.LapsCompleted.IsValue() || setter.LapsCompleted.MustGet() != 25 {
		t.Fatalf("unexpected laps_completed setter value: %+v", setter.LapsCompleted)
	}
	if !setter.FastestLapTimeMS.IsValue() || setter.FastestLapTimeMS.MustGet() != 90000 {
		t.Fatalf("unexpected fastest_lap_time_ms setter value: %+v", setter.FastestLapTimeMS)
	}
	if !setter.Incidents.IsValue() || setter.Incidents.MustGet() != 2 {
		t.Fatalf("unexpected incidents setter value: %+v", setter.Incidents)
	}
	if !setter.State.IsValue() || setter.State.MustGet() != conversion.ResultStateNormal {
		t.Fatalf("unexpected state setter value: %+v", setter.State)
	}
	if !setter.AdminNotes.IsValue() || setter.AdminNotes.MustGet() != "test note" {
		t.Fatalf("unexpected admin_notes setter value: %+v", setter.AdminNotes)
	}
}

func TestResultEntrySetterBuilderBuildDQState(t *testing.T) {
	t.Parallel()

	setter := (resultEntrySetterBuilder{}).Build(&v1.CreateResultEntryRequest{
		State: commonv1.ResultEntryState_RESULT_ENTRY_STATE_DQ,
	})

	if !setter.State.IsValue() || setter.State.MustGet() != conversion.ResultStateDQ {
		t.Fatalf("unexpected dq state setter value: %+v", setter.State)
	}
}

func TestResultEntrySetterBuilderBuildOptionalFieldsUnset(t *testing.T) {
	t.Parallel()

	setter := (resultEntrySetterBuilder{}).Build(&v1.CreateResultEntryRequest{
		RaceGridId:        0,
		DriverId:          0,
		CarModelId:        0,
		FinishingPosition: 0,
		CompletedLaps:     0,
		FastestLapTimeMs:  0,
		Incidents:         0,
		State:             commonv1.ResultEntryState_RESULT_ENTRY_STATE_UNSPECIFIED,
		AdminNotes:        "",
	})

	if setter.RaceGridID.IsValue() {
		t.Fatalf("expected race_grid_id to be unset, got: %+v", setter.RaceGridID)
	}
	if setter.DriverID.IsValue() {
		t.Fatalf("expected driver_id to be unset, got: %+v", setter.DriverID)
	}
	if setter.CarModelID.IsValue() {
		t.Fatalf("expected car_model_id to be unset, got: %+v", setter.CarModelID)
	}
	if setter.FinishPosition.IsValue() {
		t.Fatalf("expected finish_position to be unset, got: %+v", setter.FinishPosition)
	}
	if setter.LapsCompleted.IsValue() {
		t.Fatalf("expected laps_completed to be unset, got: %+v", setter.LapsCompleted)
	}
	if setter.FastestLapTimeMS.IsValue() {
		t.Fatalf("expected fastest_lap_time_ms to be unset, got: %+v", setter.FastestLapTimeMS)
	}
	if setter.Incidents.IsValue() {
		t.Fatalf("expected incidents to be unset, got: %+v", setter.Incidents)
	}
	if setter.State.IsValue() {
		t.Fatalf("expected state to be unset, got: %+v", setter.State)
	}
	if setter.AdminNotes.IsValue() {
		t.Fatalf("expected admin_notes to be unset, got: %+v", setter.AdminNotes)
	}
}

// TestCreateResultEntrySuccess tests CreateResultEntry using the real database.
func TestCreateResultEntrySuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	race := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeRace, 1)
	grid := seedRaceGrid(t, repo, race.ID, "Grid 1", conversion.RaceSessionTypeRace, 1)
	driver := seedDriver(t, repo, "ext-001", "Alex Tester")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateResultEntry(ctx, connect.NewRequest(&v1.CreateResultEntryRequest{
		RaceGridId:        uint32(grid.ID),
		DriverId:          uint32(driver.ID),
		FinishingPosition: 2,
		CompletedLaps:     20,
		State:             commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetResultEntry() == nil {
		t.Fatal("expected non-nil result entry in response")
	}
	if resp.Msg.GetResultEntry().GetRaceGridId() != uint32(grid.ID) {
		t.Fatalf(
			"unexpected race_grid_id: got %d want %d",
			resp.Msg.GetResultEntry().GetRaceGridId(),
			uint32(grid.ID),
		)
	}
	if resp.Msg.GetResultEntry().GetFinishingPosition() != 2 {
		t.Fatalf(
			"unexpected finishing_position: got %d want 2",
			resp.Msg.GetResultEntry().GetFinishingPosition(),
		)
	}
	if resp.Msg.GetResultEntry().GetState() != commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL {
		t.Fatalf("unexpected state: got %v", resp.Msg.GetResultEntry().GetState())
	}

	// Verify the created entry has the correct user set.
	id := int32(resp.Msg.GetResultEntry().GetId())
	stored, loadErr := repo.ResultEntries().LoadByID(context.Background(), id)
	if loadErr != nil {
		t.Fatalf("failed to load created result entry: %v", loadErr)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
}

// TestCreateResultEntryFailureDuplicateRaceDriver verifies that a duplicate
// (race_id, driver_id) results in CodeAlreadyExists.
func TestCreateResultEntryFailureDuplicateRaceDriver(t *testing.T) {
	repo := postgresrepo.New(testPool)
	duplicateErr := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "idx_result_entries_race_id_driver_id_unique",
	}
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return duplicateErr
		},
	})

	_, err := svc.CreateResultEntry(
		context.Background(),
		connect.NewRequest(&v1.CreateResultEntryRequest{
			RaceGridId:        1,
			DriverId:          1,
			FinishingPosition: 1,
			CompletedLaps:     20,
			State:             commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

// TestCreateResultEntrySuccessDifferentDriver verifies that the same race grid with
// different drivers succeeds using the real database.
func TestCreateResultEntrySuccessDifferentDriver(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	race := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeRace, 1)
	grid := seedRaceGrid(t, repo, race.ID, "Grid 1", conversion.RaceSessionTypeRace, 1)
	driver1 := seedDriver(t, repo, "ext-001", "Alex Tester")
	driver2 := seedDriver(t, repo, "ext-002", "Bob Racer")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	_, err := svc.CreateResultEntry(ctx, connect.NewRequest(&v1.CreateResultEntryRequest{
		RaceGridId:        uint32(grid.ID),
		DriverId:          uint32(driver1.ID),
		FinishingPosition: 1,
		CompletedLaps:     20,
		State:             commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL,
	}))
	if err != nil {
		t.Fatalf("unexpected error creating first entry: %v", err)
	}

	_, err = svc.CreateResultEntry(ctx, connect.NewRequest(&v1.CreateResultEntryRequest{
		RaceGridId:        uint32(grid.ID),
		DriverId:          uint32(driver2.ID),
		FinishingPosition: 2,
		CompletedLaps:     20,
		State:             commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL,
	}))
	if err != nil {
		t.Fatalf("unexpected error creating second entry with different driver: %v", err)
	}
}

func TestCreateResultEntryFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateResultEntry(
		context.Background(),
		connect.NewRequest(&v1.CreateResultEntryRequest{
			RaceGridId:        1,
			FinishingPosition: 1,
			CompletedLaps:     20,
			State:             commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL,
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeInternal)
	}
	if !errors.Is(err, txErr) {
		t.Fatalf("expected wrapped transaction error: %v", err)
	}
}

func TestUpdateResultEntrySuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	race := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeRace, 1)
	grid := seedRaceGrid(t, repo, race.ID, "Grid 1", conversion.RaceSessionTypeRace, 1)
	batch := seedImportBatch(t, repo, grid.ID)
	initial := seedResultEntry(t, repo, grid.ID, "Alex Tester", 1)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})
	_ = batch
	before, err := repo.ResultEntries().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial result entry: %v", err)
	}

	// Sleep briefly so UpdatedAt can advance.
	time.Sleep(10 * time.Millisecond)

	resp, err := svc.UpdateResultEntry(ctx, connect.NewRequest(&v1.UpdateResultEntryRequest{
		ResultEntryId: uint32(initial.ID),
		State:         commonv1.ResultEntryState_RESULT_ENTRY_STATE_DQ,
		AdminNotes:    "disqualified for contact",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetResultEntry() == nil {
		t.Fatal("expected non-nil result entry in response")
	}
	if resp.Msg.GetResultEntry().GetState() != commonv1.ResultEntryState_RESULT_ENTRY_STATE_DQ {
		t.Fatalf("unexpected state: got %v", resp.Msg.GetResultEntry().GetState())
	}
	if resp.Msg.GetResultEntry().GetAdminNotes() != "disqualified for contact" {
		t.Fatalf("unexpected admin_notes: got %q", resp.Msg.GetResultEntry().GetAdminNotes())
	}

	after, err := repo.ResultEntries().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated result entry: %v", err)
	}
	if after.UpdatedBy != testUserEditor {
		t.Fatalf("unexpected UpdatedBy: got %q want %q", after.UpdatedBy, testUserEditor)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf(
			"expected UpdatedAt to advance: before=%s after=%s",
			before.UpdatedAt,
			after.UpdatedAt,
		)
	}
	if after.State != conversion.ResultStateDQ {
		t.Fatalf(
			"unexpected state after update: got %q want %q",
			after.State,
			conversion.ResultStateDQ,
		)
	}
}

func TestUpdateResultEntryFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateResultEntry(
		context.Background(),
		connect.NewRequest(&v1.UpdateResultEntryRequest{
			ResultEntryId: 999999,
			State:         commonv1.ResultEntryState_RESULT_ENTRY_STATE_DQ,
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateResultEntryToDisqualified(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Automobilista 2")
	series := seedSeries(t, repo, sim.ID, "GT4")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Interlagos")
	layout := seedTrackLayout(t, repo, track.ID, "GP Layout")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 2", 2)
	race := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeRace, 1)
	grid := seedRaceGrid(t, repo, race.ID, "Grid 1", conversion.RaceSessionTypeRace, 1)
	batch := seedImportBatch(t, repo, race.ID)
	entry := seedResultEntry(t, repo, grid.ID, "Bob Racer", 3)
	_ = batch
	resp, err := svc.UpdateResultEntry(
		context.Background(),
		connect.NewRequest(&v1.UpdateResultEntryRequest{
			ResultEntryId: uint32(entry.ID),
			State:         commonv1.ResultEntryState_RESULT_ENTRY_STATE_DQ,
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetResultEntry().GetState() != commonv1.ResultEntryState_RESULT_ENTRY_STATE_DQ {
		t.Fatalf("expected dq state, got: %v", resp.Msg.GetResultEntry().GetState())
	}

	stored, loadErr := repo.ResultEntries().LoadByID(context.Background(), entry.ID)
	if loadErr != nil {
		t.Fatalf("failed to load result entry after update: %v", loadErr)
	}
	if stored.State != conversion.ResultStateDQ {
		t.Fatalf("expected state 'dq' in DB, got: %q", stored.State)
	}
}

func TestDeleteResultEntrySuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Assetto Corsa Competizione")
	series := seedSeries(t, repo, sim.ID, "GT3")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Monza")
	layout := seedTrackLayout(t, repo, track.ID, "GP Layout")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 3", 3)
	race := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeRace, 1)
	grid := seedRaceGrid(t, repo, race.ID, "Grid 1", conversion.RaceSessionTypeRace, 1)
	batch := seedImportBatch(t, repo, race.ID)
	entry := seedResultEntry(t, repo, grid.ID, "Charlie Speed", 1)
	_ = batch
	resp, err := svc.DeleteResultEntry(
		context.Background(),
		connect.NewRequest(&v1.DeleteResultEntryRequest{
			ResultEntryId: uint32(entry.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.ResultEntries().LoadByID(context.Background(), entry.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteResultEntryFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteResultEntry(
		context.Background(),
		connect.NewRequest(&v1.DeleteResultEntryRequest{
			ResultEntryId: 1,
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeInternal)
	}
	if !errors.Is(err, txErr) {
		t.Fatalf("expected wrapped transaction error: %v", err)
	}
}
