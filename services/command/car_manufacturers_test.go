//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"

	"github.com/srlmgr/backend/authn"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
)

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
