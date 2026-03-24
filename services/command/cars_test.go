//nolint:lll,dupl,funlen // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// ---------------------------------------------------------------------------
// Setter builder tests
// ---------------------------------------------------------------------------

func TestCarManufacturerSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (carManufacturerSetterBuilder{}).Build(&v1.CreateCarManufacturerRequest{
		Name: "Porsche",
	})

	if !setter.Name.IsValue() || setter.Name.MustGet() != "Porsche" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
}

func TestCarManufacturerSetterBuilderBuildZeroName(t *testing.T) {
	t.Parallel()

	setter := (carManufacturerSetterBuilder{}).Build(&v1.CreateCarManufacturerRequest{})

	if setter.Name.IsValue() {
		t.Fatalf("expected name to be unset, got: %+v", setter.Name)
	}
}

func TestCarBrandSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (carBrandSetterBuilder{}).Build(&v1.CreateCarBrandRequest{
		ManufacturerId: 5,
		Name:           "911",
	})

	if !setter.ManufacturerID.IsValue() || setter.ManufacturerID.MustGet() != 5 {
		t.Fatalf("unexpected manufacturer_id setter value: %+v", setter.ManufacturerID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "911" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
}

func TestCarModelSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (carModelSetterBuilder{}).Build(&v1.CreateCarModelRequest{
		BrandId: 3,
		Name:    "GT3 RS",
	})

	if !setter.BrandID.IsValue() || setter.BrandID.MustGet() != 3 {
		t.Fatalf("unexpected brand_id setter value: %+v", setter.BrandID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "GT3 RS" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
}

// ---------------------------------------------------------------------------
// CarManufacturer tests
// ---------------------------------------------------------------------------

func TestCreateCarManufacturerSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateCarManufacturer(ctx, connect.NewRequest(&v1.CreateCarManufacturerRequest{
		Name: "Ferrari",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarManufacturer().GetName() != "Ferrari" {
		t.Fatalf("unexpected name: %q", resp.Msg.GetCarManufacturer().GetName())
	}

	id := int32(resp.Msg.GetCarManufacturer().GetId())
	stored, err := repo.Cars().CarManufacturers().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created car manufacturer: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
}

func TestCreateCarManufacturerFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedCarManufacturer(t, repo, "BMW")

	_, err := svc.CreateCarManufacturer(
		context.Background(),
		connect.NewRequest(&v1.CreateCarManufacturerRequest{
			Name: "BMW",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreateCarManufacturerFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateCarManufacturer(
		context.Background(),
		connect.NewRequest(&v1.CreateCarManufacturerRequest{
			Name: "Audi",
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

func TestUpdateCarManufacturerSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	initial := seedCarManufacturer(t, repo, "Mercedes")
	before, err := repo.Cars().CarManufacturers().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial car manufacturer: %v", err)
	}

	resp, err := svc.UpdateCarManufacturer(ctx, connect.NewRequest(&v1.UpdateCarManufacturerRequest{
		CarManufacturerId: uint32(initial.ID),
		Name:              "Mercedes-Benz",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarManufacturer().GetName() != "Mercedes-Benz" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetCarManufacturer().GetName())
	}

	after, err := repo.Cars().CarManufacturers().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated car manufacturer: %v", err)
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
}

func TestUpdateCarManufacturerFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateCarManufacturer(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarManufacturerRequest{
			CarManufacturerId: 999,
			Name:              "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateCarManufacturerFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedCarManufacturer(t, repo, "Toyota")
	second := seedCarManufacturer(t, repo, "Honda")

	_, err := svc.UpdateCarManufacturer(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarManufacturerRequest{
			CarManufacturerId: uint32(second.ID),
			Name:              "Toyota",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Cars().CarManufacturers().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load car manufacturer after duplicate update: %v", loadErr)
	}
	if stored.Name != "Honda" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"Honda",
		)
	}
}

func TestDeleteCarManufacturerSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	initial := seedCarManufacturer(t, repo, "Delete Me Manufacturer")

	resp, err := svc.DeleteCarManufacturer(
		context.Background(),
		connect.NewRequest(&v1.DeleteCarManufacturerRequest{
			CarManufacturerId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Cars().CarManufacturers().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CarBrand tests
// ---------------------------------------------------------------------------

func seedCarBrandFixtures(t *testing.T, repo rootrepo.Repository) (manufacturerID int32) {
	t.Helper()
	cm := seedCarManufacturer(t, repo, "Fixture Manufacturer")
	return cm.ID
}

func TestCreateCarBrandSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})
	mfID := seedCarBrandFixtures(t, repo)

	resp, err := svc.CreateCarBrand(ctx, connect.NewRequest(&v1.CreateCarBrandRequest{
		ManufacturerId: uint32(mfID),
		Name:           "GT3",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarBrand().GetName() != "GT3" {
		t.Fatalf("unexpected name: %q", resp.Msg.GetCarBrand().GetName())
	}
	if resp.Msg.GetCarBrand().GetManufacturerId() != uint32(mfID) {
		t.Fatalf(
			"unexpected manufacturer_id: got %d want %d",
			resp.Msg.GetCarBrand().GetManufacturerId(),
			mfID,
		)
	}

	id := int32(resp.Msg.GetCarBrand().GetId())
	stored, err := repo.Cars().CarBrands().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created car brand: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
}

func TestCreateCarBrandFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	mfID := seedCarBrandFixtures(t, repo)
	seedCarBrand(t, repo, mfID, "GT4")

	_, err := svc.CreateCarBrand(
		context.Background(),
		connect.NewRequest(&v1.CreateCarBrandRequest{
			ManufacturerId: uint32(mfID),
			Name:           "GT4",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreateCarBrandFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateCarBrand(
		context.Background(),
		connect.NewRequest(&v1.CreateCarBrandRequest{
			ManufacturerId: 1,
			Name:           "LMP2",
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

func TestSetSimulationCarAliasesFlushAndFill(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	sim := seedSimulation(t, repo, "iRacing")
	manufacturer := seedCarManufacturer(t, repo, "Alias Manufacturer")
	brand := seedCarBrand(t, repo, manufacturer.ID, "Alias Brand")
	carModel := seedCarModel(t, repo, brand.ID, "Alias Car")
	otherCarModel := seedCarModel(t, repo, brand.ID, "Other Alias Car")

	_, err := repo.Cars().SimulationCarAliases().Create(ctx, &models.SimulationCarAliasSetter{
		CarModelID:   omit.From(carModel.ID),
		SimulationID: omit.From(sim.ID),
		ExternalName: omit.From("old-car-alias"),
		CreatedBy:    omit.From(testUserSeed),
		UpdatedBy:    omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed old car alias: %v", err)
	}

	_, err = repo.Cars().SimulationCarAliases().Create(ctx, &models.SimulationCarAliasSetter{
		CarModelID:   omit.From(otherCarModel.ID),
		SimulationID: omit.From(sim.ID),
		ExternalName: omit.From("other-car-alias"),
		CreatedBy:    omit.From(testUserSeed),
		UpdatedBy:    omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed other car alias: %v", err)
	}

	resp, err := svc.SetSimulationCarAliases(
		ctx,
		connect.NewRequest(&v1.SetSimulationCarAliasesRequest{
			CarModelId:   uint32(carModel.ID),
			SimulationId: uint32(sim.ID),
			ExternalName: []string{"new-car-alias-1"},
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetUpdated() {
		t.Fatal("expected updated=true")
	}

	_, err = repo.Cars().SimulationCarAliases().FindBySimID(ctx, sim.ID, "old-car-alias")
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected old car alias to be removed, got: %v", err)
	}

	first, err := repo.Cars().SimulationCarAliases().FindBySimID(ctx, sim.ID, "new-car-alias-1")
	if err != nil {
		t.Fatalf("failed to load first new car alias: %v", err)
	}
	if first.CarModelID != carModel.ID {
		t.Fatalf(
			"unexpected car_model_id for first alias: got %d want %d",
			first.CarModelID,
			carModel.ID,
		)
	}

	other, err := repo.Cars().SimulationCarAliases().FindBySimID(ctx, sim.ID, "other-car-alias")
	if err != nil {
		t.Fatalf("expected other car model alias to remain: %v", err)
	}
	if other.CarModelID != otherCarModel.ID {
		t.Fatalf(
			"unexpected car_model_id for other alias: got %d want %d",
			other.CarModelID,
			otherCarModel.ID,
		)
	}
}

func TestUpdateCarBrandSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})
	mfID := seedCarBrandFixtures(t, repo)

	initial := seedCarBrand(t, repo, mfID, "GTE")
	before, err := repo.Cars().CarBrands().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial car brand: %v", err)
	}

	resp, err := svc.UpdateCarBrand(ctx, connect.NewRequest(&v1.UpdateCarBrandRequest{
		CarBrandId: uint32(initial.ID),
		Name:       "GTE Pro",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarBrand().GetName() != "GTE Pro" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetCarBrand().GetName())
	}

	after, err := repo.Cars().CarBrands().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated car brand: %v", err)
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
}

func TestUpdateCarBrandFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateCarBrand(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarBrandRequest{
			CarBrandId: 999,
			Name:       "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateCarBrandFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	mfID := seedCarBrandFixtures(t, repo)
	seedCarBrand(t, repo, mfID, "GT3")
	second := seedCarBrand(t, repo, mfID, "GT4")

	_, err := svc.UpdateCarBrand(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarBrandRequest{
			CarBrandId: uint32(second.ID),
			Name:       "GT3",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Cars().CarBrands().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load car brand after duplicate update: %v", loadErr)
	}
	if stored.Name != "GT4" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"GT4",
		)
	}
}

func TestDeleteCarBrandSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	mfID := seedCarBrandFixtures(t, repo)
	initial := seedCarBrand(t, repo, mfID, "Delete Me Brand")

	resp, err := svc.DeleteCarBrand(
		context.Background(),
		connect.NewRequest(&v1.DeleteCarBrandRequest{
			CarBrandId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Cars().CarBrands().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CarModel tests
// ---------------------------------------------------------------------------

func seedCarModelFixtures(t *testing.T, repo rootrepo.Repository) (brandID int32) {
	t.Helper()
	cm := seedCarManufacturer(t, repo, "Model Fixture Manufacturer")
	cb := seedCarBrand(t, repo, cm.ID, "Model Fixture Brand")
	return cb.ID
}

func TestCreateCarModelSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})
	bID := seedCarModelFixtures(t, repo)

	resp, err := svc.CreateCarModel(ctx, connect.NewRequest(&v1.CreateCarModelRequest{
		BrandId: uint32(bID),
		Name:    "Cayman GT4",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarModel().GetName() != "Cayman GT4" {
		t.Fatalf("unexpected name: %q", resp.Msg.GetCarModel().GetName())
	}
	if resp.Msg.GetCarModel().GetBrandId() != uint32(bID) {
		t.Fatalf(
			"unexpected brand_id: got %d want %d",
			resp.Msg.GetCarModel().GetBrandId(),
			bID,
		)
	}

	id := int32(resp.Msg.GetCarModel().GetId())
	stored, err := repo.Cars().CarModels().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created car model: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
}

func TestCreateCarModelFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	bID := seedCarModelFixtures(t, repo)
	seedCarModel(t, repo, bID, "M4 GT3")

	_, err := svc.CreateCarModel(
		context.Background(),
		connect.NewRequest(&v1.CreateCarModelRequest{
			BrandId: uint32(bID),
			Name:    "M4 GT3",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreateCarModelFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateCarModel(
		context.Background(),
		connect.NewRequest(&v1.CreateCarModelRequest{
			BrandId: 1,
			Name:    "RS3 LMS",
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

func TestUpdateCarModelSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})
	bID := seedCarModelFixtures(t, repo)

	initial := seedCarModel(t, repo, bID, "992 GT3 Cup")
	before, err := repo.Cars().CarModels().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial car model: %v", err)
	}

	resp, err := svc.UpdateCarModel(ctx, connect.NewRequest(&v1.UpdateCarModelRequest{
		CarModelId: uint32(initial.ID),
		Name:       "992 GT3 Cup Updated",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarModel().GetName() != "992 GT3 Cup Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetCarModel().GetName())
	}

	after, err := repo.Cars().CarModels().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated car model: %v", err)
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
}

func TestUpdateCarModelFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateCarModel(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarModelRequest{
			CarModelId: 999,
			Name:       "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateCarModelFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	bID := seedCarModelFixtures(t, repo)
	seedCarModel(t, repo, bID, "RS5 DTM")
	second := seedCarModel(t, repo, bID, "RS3 TCR")

	_, err := svc.UpdateCarModel(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarModelRequest{
			CarModelId: uint32(second.ID),
			Name:       "RS5 DTM",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Cars().CarModels().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load car model after duplicate update: %v", loadErr)
	}
	if stored.Name != "RS3 TCR" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"RS3 TCR",
		)
	}
}

func TestDeleteCarModelSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	bID := seedCarModelFixtures(t, repo)
	initial := seedCarModel(t, repo, bID, "Delete Me Model")

	resp, err := svc.DeleteCarModel(
		context.Background(),
		connect.NewRequest(&v1.DeleteCarModelRequest{
			CarModelId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Cars().CarModels().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}
