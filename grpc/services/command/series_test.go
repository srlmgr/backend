//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
)

//nolint:whitespace // multiline signature with named return keeps lll and golines happy.
func seedSeries(
	t *testing.T,
	repo rootrepo.Repository,
	simulationID int32,
	name string,
) (
	series *models.Series,
) {
	t.Helper()

	var err error
	series, err = repo.Series().Create(context.Background(), &models.SeriesSetter{
		SimulationID: omit.From(simulationID),
		Name:         omit.From(name),
		Description:  omitnull.From("seed description"),
		IsActive:     omit.From(true),
		CreatedBy:    omit.From(testUserSeed),
		UpdatedBy:    omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed series %q: %v", name, err)
	}

	return series
}

func countSeriesRows(t *testing.T) int {
	t.Helper()

	var count int
	if err := testPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM series").
		Scan(&count); err != nil {
		t.Fatalf("failed to count series rows: %v", err)
	}

	return count
}

func TestSeriesSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (seriesSetterBuilder{}).Build(&v1.CreateSeriesRequest{
		SimulationId: 42,
		Name:         "GT3",
		Description:  "Sprint championship",
		IsActive:     true,
	})

	if !setter.SimulationID.IsValue() || setter.SimulationID.MustGet() != 42 {
		t.Fatalf("unexpected simulation_id setter value: %+v", setter.SimulationID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "GT3" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if setter.Description.IsUnset() {
		t.Fatal("expected description to be set")
	}
	if description := setter.Description.MustGetNull().
		GetOr(""); description != "Sprint championship" {
		t.Fatalf("unexpected description setter value: %q", description)
	}
	if !setter.IsActive.IsValue() || !setter.IsActive.MustGet() {
		t.Fatalf("unexpected is_active setter value: %+v", setter.IsActive)
	}
}

func TestCreateSeriesSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "rFactor 2")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateSeries(ctx, connect.NewRequest(&v1.CreateSeriesRequest{
		SimulationId: uint32(sim.ID),
		Name:         "GT Sprint",
		Description:  "Main GT series",
		IsActive:     true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetSeries().GetName() != "GT Sprint" {
		t.Fatalf("unexpected series name: %q", resp.Msg.GetSeries().GetName())
	}
	if resp.Msg.GetSeries().GetSimulationId() != uint32(sim.ID) {
		t.Fatalf(
			"unexpected simulation id: got %d want %d",
			resp.Msg.GetSeries().GetSimulationId(),
			sim.ID,
		)
	}

	id := int32(resp.Msg.GetSeries().GetId())
	stored, err := repo.Series().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created series: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if stored.SimulationID != sim.ID {
		t.Fatalf("unexpected stored simulation id: got %d want %d", stored.SimulationID, sim.ID)
	}
	if description := stored.Description.GetOr(""); description != "Main GT series" {
		t.Fatalf("unexpected stored description: %q", description)
	}
}

func TestCreateSeriesSuccessDuplicateNameDifferentSimulation(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	firstSim := seedSimulation(t, repo, "Assetto Corsa Competizione")
	secondSim := seedSimulation(t, repo, "Le Mans Ultimate")
	seedSeries(t, repo, firstSim.ID, "GT3")

	resp, err := svc.CreateSeries(
		context.Background(),
		connect.NewRequest(&v1.CreateSeriesRequest{
			SimulationId: uint32(secondSim.ID),
			Name:         "GT3",
			Description:  "Same name, different simulation",
			IsActive:     true,
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetSeries().GetSimulationId() != uint32(secondSim.ID) {
		t.Fatalf(
			"unexpected simulation id: got %d want %d",
			resp.Msg.GetSeries().GetSimulationId(),
			secondSim.ID,
		)
	}
	if got := countSeriesRows(t); got != 2 {
		t.Fatalf("unexpected series count after cross-simulation create: got %d want %d", got, 2)
	}
}

func TestCreateSeriesFailureDuplicateNameSameSimulation(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Automobilista 2")
	seedSeries(t, repo, sim.ID, "GT4")

	_, err := svc.CreateSeries(
		context.Background(),
		connect.NewRequest(&v1.CreateSeriesRequest{
			SimulationId: uint32(sim.ID),
			Name:         "GT4",
			Description:  "Duplicate within same simulation",
			IsActive:     true,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
	if got := countSeriesRows(t); got != 1 {
		t.Fatalf("unexpected series count after duplicate create: got %d want %d", got, 1)
	}
}

func TestCreateSeriesFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateSeries(
		context.Background(),
		connect.NewRequest(&v1.CreateSeriesRequest{
			SimulationId: 1,
			Name:         "Prototype",
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

func TestUpdateSeriesSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	initial := seedSeries(t, repo, sim.ID, "Porsche Cup")
	before, err := repo.Series().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial series: %v", err)
	}

	resp, err := svc.UpdateSeries(ctx, connect.NewRequest(&v1.UpdateSeriesRequest{
		SeriesId:     uint32(initial.ID),
		SimulationId: uint32(sim.ID),
		Name:         "Porsche Cup Updated",
		Description:  "Updated description",
		IsActive:     true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetSeries().GetName() != "Porsche Cup Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetSeries().GetName())
	}

	after, err := repo.Series().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated series: %v", err)
	}
	if after.UpdatedBy != testUserEditor {
		t.Fatalf("unexpected UpdatedBy: got %q want %q", after.UpdatedBy, testUserEditor)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf(
			"expected UpdatedAt to move forward: before=%s after=%s",
			before.UpdatedAt,
			after.UpdatedAt,
		)
	}
	if description := after.Description.GetOr(""); description != "Updated description" {
		t.Fatalf("unexpected updated description: %q", description)
	}
}

func TestUpdateSeriesFailureNotFound(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "RaceRoom")

	_, err := svc.UpdateSeries(
		context.Background(),
		connect.NewRequest(&v1.UpdateSeriesRequest{
			SeriesId:     999,
			SimulationId: uint32(sim.ID),
			Name:         "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateSeriesFailureDuplicateNameSameSimulation(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Automobilista 2")
	first := seedSeries(t, repo, sim.ID, "GT3")
	second := seedSeries(t, repo, sim.ID, "LMP2")

	_, err := svc.UpdateSeries(
		context.Background(),
		connect.NewRequest(&v1.UpdateSeriesRequest{
			SeriesId: uint32(second.ID),
			Name:     first.Name,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Series().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load series after duplicate update: %v", loadErr)
	}
	if stored.Name != "LMP2" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"LMP2",
		)
	}
}

func TestDeleteSeriesSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "rFactor 2")
	initial := seedSeries(t, repo, sim.ID, "Delete Me")

	resp, err := svc.DeleteSeries(
		context.Background(),
		connect.NewRequest(&v1.DeleteSeriesRequest{
			SeriesId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Series().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteSeriesFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteSeries(
		context.Background(),
		connect.NewRequest(&v1.DeleteSeriesRequest{
			SeriesId: 1,
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
