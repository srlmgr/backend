//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"

	"github.com/srlmgr/backend/authn"
	rootrepo "github.com/srlmgr/backend/repository"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
)

func TestCarModelSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (carModelSetterBuilder{}).Build(&v1.CreateCarModelRequest{
		ManufacturerId: 5,
		Name:           "911 GT3 R",
	})

	if !setter.ManufacturerID.IsValue() || setter.ManufacturerID.MustGet() != 5 {
		t.Fatalf("unexpected manufacturer_id setter value: %+v", setter.ManufacturerID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "911 GT3 R" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
}

func seedCarModelFixtures(t *testing.T, repo rootrepo.Repository) (manufacturerID int32) {
	t.Helper()
	cm := seedCarManufacturer(t, repo, "Fixture Manufacturer")
	return cm.ID
}

func TestCreateCarModelSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})
	mfID := seedCarModelFixtures(t, repo)

	resp, err := svc.CreateCarModel(ctx, connect.NewRequest(&v1.CreateCarModelRequest{
		ManufacturerId: uint32(mfID),
		Name:           "GT3",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarModel().GetName() != "GT3" {
		t.Fatalf("unexpected name: %q", resp.Msg.GetCarModel().GetName())
	}
	if resp.Msg.GetCarModel().GetManufacturerId() != uint32(mfID) {
		t.Fatalf(
			"unexpected manufacturer_id: got %d want %d",
			resp.Msg.GetCarModel().GetManufacturerId(),
			mfID,
		)
	}

	id := int32(resp.Msg.GetCarModel().GetId())
	stored, err := repo.Cars().CarModels().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created car model v2: %v", err)
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
	mfID := seedCarModelFixtures(t, repo)
	seedCarModel(t, repo, mfID, "GT4")

	_, err := svc.CreateCarModel(
		context.Background(),
		connect.NewRequest(&v1.CreateCarModelRequest{
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

func TestUpdateCarModelSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})
	mfID := seedCarModelFixtures(t, repo)

	initial := seedCarModel(t, repo, mfID, "GTE")
	before, err := repo.Cars().CarModels().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial car model v2: %v", err)
	}

	resp, err := svc.UpdateCarModel(ctx, connect.NewRequest(&v1.UpdateCarModelRequest{
		CarModelId: uint32(initial.ID),
		Name:       "GTE Pro",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarModel().GetName() != "GTE Pro" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetCarModel().GetName())
	}

	after, err := repo.Cars().CarModels().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated car model v2: %v", err)
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
	mfID := seedCarModelFixtures(t, repo)
	seedCarModel(t, repo, mfID, "GT3")
	second := seedCarModel(t, repo, mfID, "GT4")

	_, err := svc.UpdateCarModel(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarModelRequest{
			CarModelId: uint32(second.ID),
			Name:       "GT3",
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
		t.Fatalf("failed to load car model v2 after duplicate update: %v", loadErr)
	}
	if stored.Name != "GT4" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"GT4",
		)
	}
}

func TestDeleteCarModelSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	mfID := seedCarModelFixtures(t, repo)
	initial := seedCarModel(t, repo, mfID, "Delete Me Model")

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
