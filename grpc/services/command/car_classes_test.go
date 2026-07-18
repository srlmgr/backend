//nolint:dupl,lll // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/repository/repoerrors"
)

func TestCarClassSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (carClassSetterBuilder{}).Build(&v1.CreateCarClassRequest{
		Name: "GT3",
	})

	if !setter.Name.IsValue() || setter.Name.MustGet() != "GT3" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
}

func TestCarClassSetterBuilderBuildZeroName(t *testing.T) {
	t.Parallel()

	setter := (carClassSetterBuilder{}).Build(&v1.CreateCarClassRequest{})

	if setter.Name.IsValue() {
		t.Fatalf("expected name to be unset, got: %+v", setter.Name)
	}
}

func TestCreateCarClassSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateCarClass(ctx, connect.NewRequest(&v1.CreateCarClassRequest{
		Name: "GT3",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarClass().GetName() != "GT3" {
		t.Fatalf("unexpected name: %q", resp.Msg.GetCarClass().GetName())
	}

	id := int32(resp.Msg.GetCarClass().GetId())
	stored, err := repo.Cars().CarClasses().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created car class: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
}

func TestCreateCarClassFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedCarClass(t, repo, "GT4")

	_, err := svc.CreateCarClass(
		context.Background(),
		connect.NewRequest(&v1.CreateCarClassRequest{
			Name: "GT4",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestUpdateCarClassSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})
	initial := seedCarClass(t, repo, "GT3 Old")

	before, err := repo.Cars().CarClasses().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load car class before update: %v", err)
	}

	resp, err := svc.UpdateCarClass(ctx, connect.NewRequest(&v1.UpdateCarClassRequest{
		CarClassId: uint32(initial.ID),
		Name:       "GT3 New",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetCarClass().GetName() != "GT3 New" {
		t.Fatalf("unexpected name: %q", resp.Msg.GetCarClass().GetName())
	}

	after, err := repo.Cars().CarClasses().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load car class after update: %v", err)
	}
	if after.UpdatedBy != testUserEditor {
		t.Fatalf("unexpected updated_by: got %q want %q", after.UpdatedBy, testUserEditor)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf(
			"expected updated_at to advance: before=%v after=%v",
			before.UpdatedAt,
			after.UpdatedAt,
		)
	}
}

func TestUpdateCarClassFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateCarClass(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarClassRequest{
			CarClassId: 999,
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

func TestUpdateCarClassFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedCarClass(t, repo, "GT3")
	second := seedCarClass(t, repo, "GT4")

	_, err := svc.UpdateCarClass(
		context.Background(),
		connect.NewRequest(&v1.UpdateCarClassRequest{
			CarClassId: uint32(second.ID),
			Name:       "GT3",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Cars().CarClasses().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load car class after duplicate update: %v", loadErr)
	}
	if stored.Name != "GT4" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"GT4",
		)
	}
}

func TestDeleteCarClassSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	initial := seedCarClass(t, repo, "Delete Me Class")

	resp, err := svc.DeleteCarClass(
		context.Background(),
		connect.NewRequest(&v1.DeleteCarClassRequest{
			CarClassId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Cars().CarClasses().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestAssignCarModelVariantToCarClassSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	cc := seedCarClass(t, repo, "GT3")
	cm := seedCarManufacturer(t, repo, "Porsche")
	mod := seedCarModel(t, repo, cm.ID, "911")
	modVariant := seedCarModelVariant(t, repo, mod.ID, "GT3 RS")

	resp, err := svc.AssignCarModelVariantToCarClass(
		ctx,
		connect.NewRequest(&v1.AssignCarModelVariantToCarClassRequest{
			CarClassId:        uint32(cc.ID),
			CarModelVariantId: uint32(modVariant.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestAssignCarModelToCarClassTwiceIsNotAnError(t *testing.T) {
	svc, repo := newDBBackedTestService(t)

	cc := seedCarClass(t, repo, "GTC")
	cm := seedCarManufacturer(t, repo, "Ferrari")
	mod := seedCarModel(t, repo, cm.ID, "488")
	modVariant := seedCarModelVariant(t, repo, mod.ID, "488 GT3")

	_, err := svc.AssignCarModelVariantToCarClass(
		context.Background(),
		connect.NewRequest(&v1.AssignCarModelVariantToCarClassRequest{
			CarClassId:        uint32(cc.ID),
			CarModelVariantId: uint32(modVariant.ID),
		}),
	)
	if err != nil {
		t.Fatalf("first assignment failed: %v", err)
	}

	_, err = svc.AssignCarModelVariantToCarClass(
		context.Background(),
		connect.NewRequest(&v1.AssignCarModelVariantToCarClassRequest{
			CarClassId:        uint32(cc.ID),
			CarModelVariantId: uint32(modVariant.ID),
		}),
	)
	if err != nil {
		t.Fatal("error is not expected on second assignment")
	}
}
