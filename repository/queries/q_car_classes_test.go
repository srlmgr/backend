package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/repository/testhelpers"
)

//nolint:funlen // integration test seeds the required entities
func Test_queryCarClasses_FindBySeasonAndCarModelVariant(t *testing.T) {
	pool := testhelpers.NewTestPool(t, packagePool)
	queries := New(pool)
	ctx := context.Background()

	sim := testhelpers.SeedRacingSim(t, "Sim")
	series := testhelpers.SeedSeries(t, sim.ID, "Series")
	pointSystem := testhelpers.SeedPointSystem(t, "Points")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season")
	otherSeason := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Other season")
	manufacturer := testhelpers.SeedCarManufacturer(t, "Manufacturer")
	model := testhelpers.SeedCarModelV2(t, "Model", manufacturer.ID)
	variant := testhelpers.SeedCarModelVariant(t, "Variant", model.ID)

	carClass, err := models.CarClasses.Insert(&models.CarClassSetter{
		Name:      omit.From("Class"),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(testhelpers.TestUserSeed),
		UpdatedBy: omit.From(testhelpers.TestUserSeed),
	}).One(ctx, pgbob.New(pool))
	if err != nil {
		t.Fatalf("failed to seed car class: %v", err)
	}
	mapping := &models.CarClassesToCarModelSetter{
		CarClassID:        omit.From(carClass.ID),
		CarModelVariantID: omit.From(variant.ID),
	}
	_, seedErr := models.CarClassesToCarModels.Insert(mapping).One(ctx, pgbob.New(pool))
	if seedErr != nil {
		t.Fatalf("failed to associate car class with variant: %v", seedErr)
	}
	testhelpers.SeedSeasonCarClass(t, season.ID, carClass.ID, 0)

	got, err := queries.QueryCarClasses().FindBySeasonAndCarModelVariant(
		ctx,
		season.ID,
		variant.ID,
	)
	if err != nil {
		t.Fatalf("FindBySeasonAndCarModelVariant() failed: %v", err)
	}
	if got.ID != carClass.ID {
		t.Fatalf("unexpected car class: got %d want %d", got.ID, carClass.ID)
	}

	_, err = queries.QueryCarClasses().FindBySeasonAndCarModelVariant(
		ctx,
		otherSeason.ID,
		variant.ID,
	)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected car class not configured for season to be not found, got %v", err)
	}
}
