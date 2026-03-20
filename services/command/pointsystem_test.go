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

func TestPointSystemSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (pointSystemSetterBuilder{}).Build(&v1.CreatePointSystemRequest{
		Name:        "Formula Points",
		Description: "Standard formula points system",
	})

	if !setter.Name.IsValue() || setter.Name.MustGet() != "Formula Points" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.Description.IsValue() || setter.Description.MustGet() != "Standard formula points system" {
		t.Fatalf("unexpected description setter value: %+v", setter.Description)
	}
}

func TestPointSystemSetterBuilderBuildZeroValues(t *testing.T) {
	t.Parallel()

	setter := (pointSystemSetterBuilder{}).Build(&v1.CreatePointSystemRequest{})

	if setter.Name.IsValue() {
		t.Fatalf("expected name to be unset, got: %+v", setter.Name)
	}
	if setter.Description.IsValue() {
		t.Fatalf("expected description to be unset, got: %+v", setter.Description)
	}
}

func TestCreatePointSystemSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreatePointSystem(ctx, connect.NewRequest(&v1.CreatePointSystemRequest{
		Name:        "Sprint Points",
		Description: "Points for sprint races",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetPointSystem().GetName() != "Sprint Points" {
		t.Fatalf("unexpected point system name: %q", resp.Msg.GetPointSystem().GetName())
	}
	if resp.Msg.GetPointSystem().GetDescription() != "Points for sprint races" {
		t.Fatalf("unexpected point system description: %q", resp.Msg.GetPointSystem().GetDescription())
	}

	id := int32(resp.Msg.GetPointSystem().GetId())
	stored, err := repo.PointSystems().PointSystems().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created point system: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
}

func TestCreatePointSystemFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedPointSystem(t, repo, "duplicate-ps")

	_, err := svc.CreatePointSystem(
		context.Background(),
		connect.NewRequest(&v1.CreatePointSystemRequest{
			Name: "duplicate-ps",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreatePointSystemFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreatePointSystem(
		context.Background(),
		connect.NewRequest(&v1.CreatePointSystemRequest{
			Name: "tx-fail-ps",
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

func TestUpdatePointSystemSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	initial := seedPointSystem(t, repo, "Original Points")
	before, err := repo.PointSystems().PointSystems().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial point system: %v", err)
	}

	resp, err := svc.UpdatePointSystem(ctx, connect.NewRequest(&v1.UpdatePointSystemRequest{
		PointSystemId: uint32(initial.ID),
		Name:          "Updated Points",
		Description:   "Updated description",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetPointSystem().GetName() != "Updated Points" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetPointSystem().GetName())
	}

	after, err := repo.PointSystems().PointSystems().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated point system: %v", err)
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

func TestUpdatePointSystemFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdatePointSystem(
		context.Background(),
		connect.NewRequest(&v1.UpdatePointSystemRequest{
			PointSystemId: 999,
			Name:          "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdatePointSystemFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	first := seedPointSystem(t, repo, "first-ps")
	second := seedPointSystem(t, repo, "second-ps")

	_, err := svc.UpdatePointSystem(
		context.Background(),
		connect.NewRequest(&v1.UpdatePointSystemRequest{
			PointSystemId: uint32(second.ID),
			Name:          first.Name,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.PointSystems().PointSystems().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load point system after duplicate update: %v", loadErr)
	}
	if stored.Name != "second-ps" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"second-ps",
		)
	}
}

func TestDeletePointSystemSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	initial := seedPointSystem(t, repo, "delete-me-ps")

	resp, err := svc.DeletePointSystem(
		context.Background(),
		connect.NewRequest(&v1.DeletePointSystemRequest{
			PointSystemId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.PointSystems().PointSystems().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeletePointSystemFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeletePointSystem(
		context.Background(),
		connect.NewRequest(&v1.DeletePointSystemRequest{
			PointSystemId: 1,
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
