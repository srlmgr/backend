package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/repository/seasons"
	"github.com/srlmgr/backend/repository/testhelpers"
)

func Test_querySeasonCarModelVariants_FindBySeasonAndCarModelVariant(t *testing.T) {
	pool := testhelpers.NewTestPool(t, packagePool)
	repo := New(pool)
	sim := testhelpers.SeedRacingSim(t, "Sim")
	series := testhelpers.SeedSeries(t, sim.ID, "Series")
	pointSystem := testhelpers.SeedPointSystem(t, "Points")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season")
	manufacturer := testhelpers.SeedCarManufacturer(t, "Manufacturer")
	model := testhelpers.SeedCarModelV2(t, "Model", manufacturer.ID)
	included := testhelpers.SeedCarModelVariant(t, "Included", model.ID)
	excluded := testhelpers.SeedCarModelVariant(t, "Excluded", model.ID)
	if err := seasons.New(pool).SetCarModelVariants(
		context.Background(),
		season.ID,
		[]int32{included.ID},
	); err != nil {
		t.Fatalf("failed to set season car model variants: %v", err)
	}

	got, err := repo.QuerySeasonCarModelVariants().FindBySeasonAndCarModelVariant(
		context.Background(),
		season.ID,
		included.ID,
	)
	if err != nil {
		t.Fatalf("FindBySeasonAndCarModelVariant() failed: %v", err)
	}
	if got.CarModelVariantID != included.ID {
		t.Fatalf(
			"unexpected car model variant: got %d want %d",
			got.CarModelVariantID,
			included.ID,
		)
	}

	_, err = repo.QuerySeasonCarModelVariants().FindBySeasonAndCarModelVariant(
		context.Background(),
		season.ID,
		excluded.ID,
	)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected excluded variant to be not found, got %v", err)
	}
}
