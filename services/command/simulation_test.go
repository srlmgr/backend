//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"os"
	"testing"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/testsupport/testdb"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := testdb.InitTestDB()
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	testPool = pool
	code := m.Run()
	testPool.Close()
	os.Exit(code)
}

type txManagerStub struct {
	runInTx func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (t txManagerStub) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if t.runInTx != nil {
		return t.runInTx(ctx, fn)
	}
	return fn(ctx)
}

func newTestService(repo rootrepo.Repository, txMgr rootrepo.TransactionManager) *service {
	return &service{
		logger:     log.New(),
		repo:       repo,
		txMgr:      txMgr,
		conversion: conversion.New(),
	}
}

func newDBBackedTestService(t *testing.T) (*service, rootrepo.Repository) {
	t.Helper()
	resetRacingSimsTable(t)
	t.Cleanup(func() {
		resetRacingSimsTable(t)
	})

	repo := postgresrepo.New(testPool)
	txMgr := rootrepo.NewBobTransactionFromPool(testPool)

	return newTestService(repo, txMgr), repo
}

func resetRacingSimsTable(t *testing.T) {
	t.Helper()

	if _, err := testPool.Exec(
		context.Background(),
		"TRUNCATE TABLE racing_sims RESTART IDENTITY CASCADE",
	); err != nil {
		t.Fatalf("failed to reset racing_sims table: %v", err)
	}
}

func seedSimulation(t *testing.T, repo rootrepo.Repository, name string) *models.RacingSim {
	t.Helper()

	sim, err := repo.RacingSims().Create(context.Background(), &models.RacingSimSetter{
		Name:                   omit.From(name),
		IsActive:               omit.From(true),
		SupportedImportFormats: omit.From(pq.StringArray{"json"}),
		CreatedBy:              omit.From("seed"),
		UpdatedBy:              omit.From("seed"),
	})
	if err != nil {
		t.Fatalf("failed to seed simulation %q: %v", name, err)
	}

	return sim
}

func TestRacingSimSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter, err := (racingSimSetterBuilder{}).Build(&v1.CreateSimulationRequest{
		Name:     "Le Mans Ultimate",
		IsActive: true,
		SupportedFormats: []commonv1.ImportFormat{
			commonv1.ImportFormat_IMPORT_FORMAT_JSON,
			commonv1.ImportFormat_IMPORT_FORMAT_CSV,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "Le Mans Ultimate" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.IsActive.IsValue() || !setter.IsActive.MustGet() {
		t.Fatalf("unexpected is_active setter value: %+v", setter.IsActive)
	}
	if !setter.SupportedImportFormats.IsValue() {
		t.Fatal("expected supported import formats to be set")
	}

	formats := setter.SupportedImportFormats.MustGet()
	if len(formats) != 2 || formats[0] != "json" || formats[1] != "csv" {
		t.Fatalf("unexpected supported formats: %v", formats)
	}
}

func TestRacingSimSetterBuilderBuildFailureInvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := (racingSimSetterBuilder{}).Build(&v1.CreateSimulationRequest{
		SupportedFormats: []commonv1.ImportFormat{commonv1.ImportFormat(99)},
	})
	if err == nil {
		t.Fatal("expected error for invalid import format")
	}
}

func TestCreateSimulationSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: "tester"})

	resp, err := svc.CreateSimulation(ctx, connect.NewRequest(&v1.CreateSimulationRequest{
		Name:             "rFactor 2",
		IsActive:         true,
		SupportedFormats: []commonv1.ImportFormat{commonv1.ImportFormat_IMPORT_FORMAT_JSON},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetSimulation().GetName() != "rFactor 2" {
		t.Fatalf("unexpected simulation name: %q", resp.Msg.GetSimulation().GetName())
	}

	id := int32(resp.Msg.GetSimulation().GetId())
	stored, err := repo.RacingSims().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created simulation: %v", err)
	}
	if stored.CreatedBy != "tester" || stored.UpdatedBy != "tester" {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if len(stored.SupportedImportFormats) != 1 || stored.SupportedImportFormats[0] != "json" {
		t.Fatalf("unexpected stored formats: %v", stored.SupportedImportFormats)
	}
}

func TestCreateSimulationFailureInvalidFormat(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.CreateSimulation(
		context.Background(),
		connect.NewRequest(&v1.CreateSimulationRequest{
			SupportedFormats: []commonv1.ImportFormat{commonv1.ImportFormat(99)},
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeInvalidArgument)
	}
}

func TestCreateSimulationFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedSimulation(t, repo, "duplicate-sim")

	_, err := svc.CreateSimulation(
		context.Background(),
		connect.NewRequest(&v1.CreateSimulationRequest{
			Name:             "duplicate-sim",
			IsActive:         true,
			SupportedFormats: []commonv1.ImportFormat{commonv1.ImportFormat_IMPORT_FORMAT_JSON},
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	items, loadErr := repo.RacingSims().LoadAll(context.Background())
	if loadErr != nil {
		t.Fatalf("failed to load simulations after duplicate create: %v", loadErr)
	}
	if len(items) != 1 {
		t.Fatalf(
			"unexpected simulation count after duplicate create: got %d want %d",
			len(items),
			1,
		)
	}
}

func TestCreateSimulationFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New("transaction failed")
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateSimulation(
		context.Background(),
		connect.NewRequest(&v1.CreateSimulationRequest{
			Name: "ACC",
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

func TestUpdateSimulationSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: "editor"})

	initial := seedSimulation(t, repo, "iRacing")
	before, err := repo.RacingSims().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial simulation: %v", err)
	}

	resp, err := svc.UpdateSimulation(ctx, connect.NewRequest(&v1.UpdateSimulationRequest{
		SimulationId:     uint32(initial.ID),
		Name:             "iRacing Updated",
		IsActive:         true,
		SupportedFormats: []commonv1.ImportFormat{commonv1.ImportFormat_IMPORT_FORMAT_CSV},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetSimulation().GetName() != "iRacing Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetSimulation().GetName())
	}

	after, err := repo.RacingSims().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated simulation: %v", err)
	}
	if after.UpdatedBy != "editor" {
		t.Fatalf("unexpected UpdatedBy: got %q want %q", after.UpdatedBy, "editor")
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf(
			"expected UpdatedAt to move forward: before=%s after=%s",
			before.UpdatedAt,
			after.UpdatedAt,
		)
	}
	if len(after.SupportedImportFormats) != 1 || after.SupportedImportFormats[0] != "csv" {
		t.Fatalf("unexpected updated formats: %v", after.SupportedImportFormats)
	}
}

func TestUpdateSimulationFailureInvalidFormat(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	initial := seedSimulation(t, repo, "invalid-format-target")

	_, err := svc.UpdateSimulation(
		context.Background(),
		connect.NewRequest(&v1.UpdateSimulationRequest{
			SimulationId:     uint32(initial.ID),
			SupportedFormats: []commonv1.ImportFormat{commonv1.ImportFormat(99)},
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeInvalidArgument)
	}
}

func TestUpdateSimulationFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateSimulation(
		context.Background(),
		connect.NewRequest(&v1.UpdateSimulationRequest{
			SimulationId: 999,
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

func TestUpdateSimulationFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	first := seedSimulation(t, repo, "first-sim")
	second := seedSimulation(t, repo, "second-sim")

	_, err := svc.UpdateSimulation(
		context.Background(),
		connect.NewRequest(&v1.UpdateSimulationRequest{
			SimulationId: uint32(second.ID),
			Name:         first.Name,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.RacingSims().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load simulation after duplicate update: %v", loadErr)
	}
	if stored.Name != "second-sim" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"second-sim",
		)
	}
}

func TestDeleteSimulationSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	initial := seedSimulation(t, repo, "delete-me")

	resp, err := svc.DeleteSimulation(
		context.Background(),
		connect.NewRequest(&v1.DeleteSimulationRequest{
			SimulationId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.RacingSims().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteSimulationFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New("transaction failed")
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteSimulation(
		context.Background(),
		connect.NewRequest(&v1.DeleteSimulationRequest{
			SimulationId: 1,
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
