//nolint:dupl,funlen // test code
package queries

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/repository/testhelpers"
)

func Test_querySeasonDrivers_ResolveSeasonDriver(t *testing.T) {
	repo := newDBBackedRepository(t)
	sim := testhelpers.SeedRacingSim(t, "Sim A")
	series := testhelpers.SeedSeries(t, sim.ID, "Series A")
	pointSystem := testhelpers.SeedPointSystem(t, "Point System A")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season A")
	driver := testhelpers.SeedDriver(t, "Driver A", "extA")
	carManufacturer := testhelpers.SeedCarManufacturer(t, "Audi")
	carBrand := testhelpers.SeedCarBrand(t, "Audi R8", carManufacturer.ID)
	carModelA := testhelpers.SeedCarModel(t, "Audi R8 LMS", carBrand.ID)
	carModelB := testhelpers.SeedCarModel(t, "Audi R8 LMS EVO 2", carBrand.ID)
	seasonDriverA := testhelpers.SeedSeasonDriver(
		t,
		"12",
		driver.ID,
		season.ID,
		carModelA.ID,
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		new(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)),
	)
	seasonDriverB := testhelpers.SeedSeasonDriver(
		t,
		"24",
		driver.ID,
		season.ID,
		carModelB.ID,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		nil,
	)
	_ = carModelB

	tests := []struct {
		name     string // description of this test case
		seasonID int32
		driverID int32
		when     time.Time
		want     *models.SeasonDriver
		wantErr  bool
	}{
		{
			name: "before any", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), wantErr: true,
		},
		{
			name: "in phase 1 (left edge)", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), want: seasonDriverA,
		},
		{
			name: "in phase 2 (left edge)", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), want: seasonDriverB,
		},
		{
			name: "unknown driver id", seasonID: season.ID, driverID: int32(999),
			when: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), wantErr: true,
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := repo.QuerySeasonDrivers().
				ResolveSeasonDriver(ctx, tt.seasonID, tt.driverID, tt.when)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ResolveSeasonDriver() failed: %v", gotErr)
				}
				if !errors.Is(gotErr, repoerrors.ErrNotFound) {
					t.Errorf("ResolveSeasonDriver() failed with unexpected error: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ResolveSeasonDriver() succeeded unexpectedly")
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveSeasonDriver() = %v, want %v", got, tt.want)
			}
		})
	}
}
