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

func TestCarModelVariantSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (carModelVariantSetterBuilder{}).Build(&v1.CreateCarModelVariantRequest{
		ModelId: 3,
		Name:    "GT3 R Evo",
	})

	if !setter.CarModelID.IsValue() || setter.CarModelID.MustGet() != 3 {
		t.Fatalf("unexpected model_id setter value: %+v", setter.CarModelID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "GT3 R Evo" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
}

func seedCarModelVariantFixtures(t *testing.T, repo rootrepo.Repository) (modelID int32) {
	t.Helper()
	cm := seedCarManufacturer(t, repo, "Model Variant Fixture Manufacturer")
	cmod := seedCarModel(t, repo, cm.ID, "Model Variant Fixture Model")
	return cmod.ID
}

func TestSetSimulationCarAliasesFlushAndFill(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	sim := seedSimulation(t, repo, "iRacing")
	manufacturer := seedCarManufacturer(t, repo, "Alias Manufacturer")
	carModel := seedCarModel(t, repo, manufacturer.ID, "Alias Car Model")
	carModelVariant := seedCarModelVariant(t, repo, carModel.ID, "Alias Car Model Variant")
	otherModel := seedCarModel(t, repo, manufacturer.ID, "Other Alias Car Model")
	otherModelVariant := seedCarModelVariant(
		t,
		repo,
		otherModel.ID,
		"Other Alias Car Model Variant",
	)

	_, err := repo.Cars().SimulationCarAliases().Create(ctx, &models.SimulationCarAliasSetter{
		CarModelVariantID: omit.From(carModelVariant.ID),
		SimulationID:      omit.From(sim.ID),
		ExternalName:      omit.From("old-car-alias"),
		CreatedBy:         omit.From(testUserSeed),
		UpdatedBy:         omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed old car alias: %v", err)
	}

	_, err = repo.Cars().SimulationCarAliases().Create(ctx, &models.SimulationCarAliasSetter{
		CarModelVariantID: omit.From(otherModelVariant.ID),
		SimulationID:      omit.From(sim.ID),
		ExternalName:      omit.From("other-car-alias"),
		CreatedBy:         omit.From(testUserSeed),
		UpdatedBy:         omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed other car alias: %v", err)
	}

	resp, err := svc.SetSimulationCarAliases(
		ctx,
		connect.NewRequest(&v1.SetSimulationCarAliasesRequest{
			CarModelVariantId: uint32(carModelVariant.ID),
			SimulationId:      uint32(sim.ID),
			ExternalName:      []string{"new-car-alias-1"},
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
	if first.CarModelVariantID != carModelVariant.ID {
		t.Fatalf(
			"unexpected car_model_variant_id for first alias: got %d want %d",
			first.CarModelVariantID,
			carModelVariant.ID,
		)
	}

	other, err := repo.Cars().SimulationCarAliases().FindBySimID(ctx, sim.ID, "other-car-alias")
	if err != nil {
		t.Fatalf("expected other car model alias to remain: %v", err)
	}
	if other.CarModelVariantID != otherModelVariant.ID {
		t.Fatalf(
			"unexpected car_model_variant_id for other alias: got %d want %d",
			other.CarModelVariantID,
			otherModelVariant.ID,
		)
	}
}

func TestCreateCarModelVariantSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})
	modelID := seedCarModelVariantFixtures(t, repo)

	resp, err := svc.CreateCarModelVariant(
		ctx,
		connect.NewRequest(&v1.CreateCarModelVariantRequest{
			ModelId: uint32(modelID),
			Name:    "Cayman GT4",
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarModelVariant().GetName() != "Cayman GT4" {
		t.Fatalf("unexpected name: %q", resp.Msg.GetCarModelVariant().GetName())
	}
	if resp.Msg.GetCarModelVariant().GetModelId() != uint32(modelID) {
		t.Fatalf(
			"unexpected model_id: got %d want %d",
			resp.Msg.GetCarModelVariant().GetModelId(),
			modelID,
		)
	}

	id := int32(resp.Msg.GetCarModelVariant().GetId())
	stored, err := repo.Cars().CarModelVariants().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created car model variant: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
}

func TestCreateCarModelVariantFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	modelID := seedCarModelVariantFixtures(t, repo)
	seedCarModelVariant(t, repo, modelID, "M4 GT3")

	_, err := svc.CreateCarModelVariant(
		context.Background(),
		connect.NewRequest(&v1.CreateCarModelVariantRequest{
			ModelId: uint32(modelID),
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

func TestCreateCarModelVariantFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateCarModelVariant(
		context.Background(),
		connect.NewRequest(&v1.CreateCarModelVariantRequest{
			ModelId: 1,
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

func TestUpdateCarModelVariantSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})
	modelID := seedCarModelVariantFixtures(t, repo)

	initial := seedCarModelVariant(t, repo, modelID, "992 GT3 Cup")
	before, err := repo.Cars().CarModelVariants().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial car model variant: %v", err)
	}

	resp, err := svc.UpdateCarModelVariant(
		ctx,
		connect.NewRequest(&v1.UpdateCarModelVariantRequest{
			CarModelVariantId: uint32(initial.ID),
			Name:              "992 GT3 Cup Updated",
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarModelVariant().GetName() != "992 GT3 Cup Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetCarModelVariant().GetName())
	}

	after, err := repo.Cars().CarModelVariants().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated car model variant: %v", err)
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

func TestUpdateCarModelVariantFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateCarModelVariant(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarModelVariantRequest{
			CarModelVariantId: 999,
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

func TestUpdateCarModelVariantFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	modelID := seedCarModelVariantFixtures(t, repo)
	seedCarModelVariant(t, repo, modelID, "RS5 DTM")
	second := seedCarModelVariant(t, repo, modelID, "RS3 TCR")

	_, err := svc.UpdateCarModelVariant(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarModelVariantRequest{
			CarModelVariantId: uint32(second.ID),
			Name:              "RS5 DTM",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Cars().CarModelVariants().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load car model variant after duplicate update: %v", loadErr)
	}
	if stored.Name != "RS3 TCR" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"RS3 TCR",
		)
	}
}

func TestDeleteCarModelVariantSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	modelID := seedCarModelVariantFixtures(t, repo)
	initial := seedCarModelVariant(t, repo, modelID, "Delete Me Model")

	resp, err := svc.DeleteCarModelVariant(
		context.Background(),
		connect.NewRequest(&v1.DeleteCarModelVariantRequest{
			CarModelVariantId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Cars().CarModelVariants().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}
