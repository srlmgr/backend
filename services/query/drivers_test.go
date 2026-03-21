//nolint:dupl // test files can have some duplication for test data setup
package query

import (
	"context"
	"errors"
	"testing"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
)

//nolint:whitespace // editor/linter issue
func seedDriver(
	t *testing.T,
	repo rootrepo.Repository,
	externalID string,
	name string,
) *models.Driver {
	t.Helper()
	d, err := repo.Drivers().Drivers().Create(context.Background(), &models.DriverSetter{
		ExternalID: omit.From(externalID),
		Name:       omit.From(name),
		IsActive:   omit.From(true),
		CreatedBy:  omit.From(testUserSeed),
		UpdatedBy:  omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed driver %q: %v", name, err)
	}
	return d
}

func TestListDriversEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListDrivers(
		context.Background(),
		connect.NewRequest(&queryv1.ListDriversRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListDriversReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	alpha := seedDriver(t, repo, "1001", "Alpha Driver")
	beta := seedDriver(t, repo, "1002", "Beta Driver")

	resp, err := svc.ListDrivers(
		context.Background(),
		connect.NewRequest(&queryv1.ListDriversRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	ids := make(map[uint32]bool)
	for _, item := range items {
		ids[item.GetId()] = true
	}

	if !ids[uint32(alpha.ID)] {
		t.Errorf("alpha driver (id=%d) not found in response", alpha.ID)
	}
	if !ids[uint32(beta.ID)] {
		t.Errorf("beta driver (id=%d) not found in response", beta.ID)
	}
}

func TestGetDriverSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	d := seedDriver(t, repo, "2001", "Test Driver")

	resp, err := svc.GetDriver(
		context.Background(),
		connect.NewRequest(&queryv1.GetDriverRequest{
			Id: uint32(d.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := resp.Msg.GetDriver()
	if got.GetId() != uint32(d.ID) {
		t.Errorf("expected id %d, got %d", d.ID, got.GetId())
	}
	if got.GetExternalId() != "2001" {
		t.Errorf("expected external_id %q, got %q", "2001", got.GetExternalId())
	}
	if got.GetName() != "Test Driver" {
		t.Errorf("expected name %q, got %q", "Test Driver", got.GetName())
	}
	if !got.GetIsActive() {
		t.Errorf("expected is_active true, got false")
	}
}

func TestGetDriverNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetDriver(
		context.Background(),
		connect.NewRequest(&queryv1.GetDriverRequest{
			Id: 99999,
		}),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}
