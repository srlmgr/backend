//nolint:lll,dupl // test code can be verbose
package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/srlmgr/backend/repository/repoerrors"
)

func TestDriverSimulationIDsFindBySimID(t *testing.T) {
	t.Parallel()

	repo := New()
	ctx := context.Background()

	item, err := repo.Drivers().DriverSimulationIDs().FindBySimID(ctx, 1, "alex-ir-01")
	if err != nil {
		t.Fatalf("FindBySimID returned error: %v", err)
	}
	if item == nil {
		t.Fatal("FindBySimID returned nil item")
	}
	if item.ID != 1 {
		t.Fatalf("unexpected item id: got %d want 1", item.ID)
	}

	_, err = repo.Drivers().DriverSimulationIDs().FindBySimID(ctx, 2, "alex-ir-01")
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for wrong simulation, got %v", err)
	}
}

func TestSimulationCarAliasesFindBySimID(t *testing.T) {
	t.Parallel()

	repo := New()
	ctx := context.Background()

	item, err := repo.Cars().SimulationCarAliases().FindBySimID(ctx, 1, "Porsche 911 GT3 R")
	if err != nil {
		t.Fatalf("FindBySimID returned error: %v", err)
	}
	if item == nil {
		t.Fatal("FindBySimID returned nil item")
	}
	if item.ID != 1 {
		t.Fatalf("unexpected item id: got %d want 1", item.ID)
	}

	_, err = repo.Cars().SimulationCarAliases().FindBySimID(ctx, 1, "missing")
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing alias, got %v", err)
	}
}
