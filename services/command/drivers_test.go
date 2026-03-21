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

func TestDriverSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (driverSetterBuilder{}).Build(&v1.CreateDriverRequest{
		ExternalId: "42",
		Name:       "Max Verstappen",
		IsActive:   true,
	})

	if !setter.ExternalID.IsValue() || setter.ExternalID.MustGet() != "42" {
		t.Fatalf("unexpected external_id setter value: %+v", setter.ExternalID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "Max Verstappen" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.IsActive.IsValue() || !setter.IsActive.MustGet() {
		t.Fatalf("unexpected is_active setter value: %+v", setter.IsActive)
	}
}

func TestDriverSetterBuilderBuildZeroValues(t *testing.T) {
	t.Parallel()

	setter := (driverSetterBuilder{}).Build(&v1.CreateDriverRequest{
		ExternalId: "",
		Name:       "",
		IsActive:   false,
	})

	if setter.ExternalID.IsValue() {
		t.Fatalf("expected external_id to be unset when zero, got: %+v", setter.ExternalID)
	}
	if setter.Name.IsValue() {
		t.Fatalf("expected name to be unset when empty, got: %+v", setter.Name)
	}
	if setter.IsActive.IsValue() {
		t.Fatalf("expected is_active to be unset when false, got: %+v", setter.IsActive)
	}
}

func TestCreateDriverSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateDriver(ctx, connect.NewRequest(&v1.CreateDriverRequest{
		ExternalId: "99",
		Name:       "Lewis Hamilton",
		IsActive:   true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetDriver().GetName() != "Lewis Hamilton" {
		t.Fatalf("unexpected driver name: %q", resp.Msg.GetDriver().GetName())
	}
	if resp.Msg.GetDriver().GetExternalId() != "99" {
		t.Fatalf(
			"unexpected external id: got %q want %q",
			resp.Msg.GetDriver().GetExternalId(),
			"99",
		)
	}

	id := int32(resp.Msg.GetDriver().GetId())
	stored, err := repo.Drivers().Drivers().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created driver: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if stored.ExternalID != "99" {
		t.Fatalf("unexpected stored external_id: got %q want %q", stored.ExternalID, "99")
	}
}

func TestCreateDriverFailureDuplicateExternalID(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedDriver(t, repo, "100", "Fernando Alonso")

	_, err := svc.CreateDriver(
		context.Background(),
		connect.NewRequest(&v1.CreateDriverRequest{
			ExternalId: "100",
			Name:       "Carlos Sainz",
			IsActive:   true,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreateDriverFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateDriver(
		context.Background(),
		connect.NewRequest(&v1.CreateDriverRequest{
			ExternalId: "1",
			Name:       "Test Driver",
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

func TestUpdateDriverSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	initial := seedDriver(t, repo, "200", "Sebastian Vettel")
	before, err := repo.Drivers().Drivers().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial driver: %v", err)
	}

	resp, err := svc.UpdateDriver(ctx, connect.NewRequest(&v1.UpdateDriverRequest{
		DriverId:   uint32(initial.ID),
		ExternalId: "200",
		Name:       "Sebastian Vettel Updated",
		IsActive:   true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetDriver().GetName() != "Sebastian Vettel Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetDriver().GetName())
	}

	after, err := repo.Drivers().Drivers().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated driver: %v", err)
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

func TestUpdateDriverFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateDriver(
		context.Background(),
		connect.NewRequest(&v1.UpdateDriverRequest{
			DriverId: 999,
			Name:     "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateDriverFailureDuplicateExternalID(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedDriver(t, repo, "300", "Kimi Raikkonen")
	second := seedDriver(t, repo, "301", "Valtteri Bottas")

	_, err := svc.UpdateDriver(
		context.Background(),
		connect.NewRequest(&v1.UpdateDriverRequest{
			DriverId:   uint32(second.ID),
			ExternalId: "300",
			Name:       second.Name,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Drivers().Drivers().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load driver after duplicate update: %v", loadErr)
	}
	if stored.Name != "Valtteri Bottas" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"Valtteri Bottas",
		)
	}
}

func TestDeleteDriverSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	initial := seedDriver(t, repo, "400", "Delete Me Driver")

	resp, err := svc.DeleteDriver(
		context.Background(),
		connect.NewRequest(&v1.DeleteDriverRequest{
			DriverId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Drivers().Drivers().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteDriverFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteDriver(
		context.Background(),
		connect.NewRequest(&v1.DeleteDriverRequest{
			DriverId: 1,
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
