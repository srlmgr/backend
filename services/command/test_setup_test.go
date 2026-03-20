package command

import (
	"context"
	"os"
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/testsupport/testdb"
)

const (
	testUserSeed   = "seed"
	testUserTester = "tester"
	testUserEditor = "editor"

	txFailedErrMsg = "transaction failed"
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
	runInTx func(
		ctx context.Context,
		fn func(ctx context.Context) error,
	) error
}

//nolint:whitespace // multiline signature style
func (t txManagerStub) RunInTx(
	ctx context.Context,
	fn func(ctx context.Context) error,
) (
	err error,
) {
	if t.runInTx != nil {
		return t.runInTx(ctx, fn)
	}
	return fn(ctx)
}

//nolint:whitespace // multiline signature style
func newTestService(
	repo rootrepo.Repository,
	txMgr rootrepo.TransactionManager,
) (
	svc *service,
) {
	svc = &service{
		logger:     log.New(),
		repo:       repo,
		txMgr:      txMgr,
		conversion: conversion.New(),
	}

	return svc
}

func newDBBackedTestService(t *testing.T) (*service, rootrepo.Repository) {
	t.Helper()
	resetTestTables(t)
	t.Cleanup(func() {
		resetTestTables(t)
	})

	repo := postgresrepo.New(testPool)
	txMgr := rootrepo.NewBobTransactionFromPool(testPool)

	return newTestService(repo, txMgr), repo
}

func resetTestTables(t *testing.T) {
	t.Helper()

	if _, err := testPool.Exec(
		context.Background(),
		"TRUNCATE TABLE car_models, car_brands, car_manufacturers, racing_sims, point_systems RESTART IDENTITY CASCADE",
	); err != nil {
		t.Fatalf("failed to reset test tables: %v", err)
	}
}

//nolint:whitespace // multiline signature style
func seedSimulation(
	t *testing.T,
	repo rootrepo.Repository,
	name string,
) (
	sim *models.RacingSim,
) {
	t.Helper()

	var err error
	sim, err = repo.RacingSims().Create(context.Background(), &models.RacingSimSetter{
		Name:                   omit.From(name),
		IsActive:               omit.From(true),
		SupportedImportFormats: omit.From(pq.StringArray{"json"}),
		CreatedBy:              omit.From(testUserSeed),
		UpdatedBy:              omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed simulation %q: %v", name, err)
	}

	return sim
}

//nolint:whitespace // multiline signature style
func seedPointSystem(
	t *testing.T,
	repo rootrepo.Repository,
	name string,
) (
	ps *models.PointSystem,
) {
	t.Helper()

	var err error
	ps, err = repo.PointSystems().PointSystems().Create(context.Background(), &models.PointSystemSetter{
		Name:      omit.From(name),
		CreatedBy: omit.From(testUserSeed),
		UpdatedBy: omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed point system %q: %v", name, err)
	}

	return ps
}

//nolint:whitespace // multiline signature style
func seedCarManufacturer(
	t *testing.T,
	repo rootrepo.Repository,
	name string,
) (
	cm *models.CarManufacturer,
) {
	t.Helper()

	var err error
	cm, err = repo.Cars().CarManufacturers().Create(context.Background(), &models.CarManufacturerSetter{
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(testUserSeed),
		UpdatedBy: omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car manufacturer %q: %v", name, err)
	}

	return cm
}

//nolint:whitespace // multiline signature style
func seedCarBrand(
	t *testing.T,
	repo rootrepo.Repository,
	manufacturerID int32,
	name string,
) (
	cb *models.CarBrand,
) {
	t.Helper()

	var err error
	cb, err = repo.Cars().CarBrands().Create(context.Background(), &models.CarBrandSetter{
		ManufacturerID: omit.From(manufacturerID),
		Name:           omit.From(name),
		IsActive:       omit.From(true),
		CreatedBy:      omit.From(testUserSeed),
		UpdatedBy:      omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car brand %q: %v", name, err)
	}

	return cb
}

//nolint:whitespace // multiline signature style
func seedCarModel(
	t *testing.T,
	repo rootrepo.Repository,
	brandID int32,
	name string,
) (
	cmod *models.CarModel,
) {
	t.Helper()

	var err error
	cmod, err = repo.Cars().CarModels().Create(context.Background(), &models.CarModelSetter{
		BrandID:   omit.From(brandID),
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(testUserSeed),
		UpdatedBy: omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car model %q: %v", name, err)
	}

	return cmod
}
