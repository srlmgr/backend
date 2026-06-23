//nolint:dupl,funlen,lll // test code
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

func Test_queryTeamDrivers_ResolveTeamDriver(t *testing.T) {
	teamJoinedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	driverJoinedAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	repo := newDBBackedRepository(t)
	sim := testhelpers.SeedRacingSim(t, "Sim A")
	series := testhelpers.SeedSeries(t, sim.ID, "Series A")
	pointSystem := testhelpers.SeedPointSystem(t, "Point System A")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season A")
	driver := testhelpers.SeedDriver(t, "Driver A", "extA")
	team := testhelpers.SeedTeamSimple(t, season.ID, "Team A", teamJoinedAt, nil)
	teamDriver := testhelpers.SeedTeamDriver(t, team.ID, driver.ID, driverJoinedAt, nil)

	tests := []struct {
		name     string // description of this test case
		seasonID int32
		driverID int32
		when     time.Time
		want     *models.TeamDriver
		wantErr  bool
	}{
		{
			name: "after both join dates", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), want: teamDriver,
		},
		{
			name: "before both join dates", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), wantErr: true,
		},
		{
			name: "unknown driver id", seasonID: season.ID, driverID: int32(999),
			when: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), wantErr: true,
		},
		{
			name: "before driver joined team", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), wantErr: true,
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := repo.QueryTeamDrivers().
				ResolveTeamDriver(ctx, tt.seasonID, tt.driverID, tt.when)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ResolveTeamDriver() failed: %v", gotErr)
				}
				if !errors.Is(gotErr, repoerrors.ErrNotFound) {
					t.Errorf("ResolveTeamDriver() failed with unexpected error: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ResolveTeamDriver() succeeded unexpectedly")
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveTeamDriver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryTeamDrivers_ResolveTeamDriverWithTeamChanges(t *testing.T) {
	teamJoinedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	driverJoinedTeamA := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	driverLeftTeamA := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	driverJoinedTeamB := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	driverLeftTeamB := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	driverJoinedTeamC := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	repo := newDBBackedRepository(t)
	sim := testhelpers.SeedRacingSim(t, "Sim A")
	series := testhelpers.SeedSeries(t, sim.ID, "Series A")
	pointSystem := testhelpers.SeedPointSystem(t, "Point System A")
	season := testhelpers.SeedSeason(t, series.ID, pointSystem.ID, "Season A")
	driver := testhelpers.SeedDriver(t, "Driver A", "extA")
	teamA := testhelpers.SeedTeamSimple(t, season.ID, "Team A", teamJoinedAt, nil)
	teamB := testhelpers.SeedTeamSimple(t, season.ID, "Team B", teamJoinedAt, nil)
	teamC := testhelpers.SeedTeamSimple(t, season.ID, "Team C", teamJoinedAt,
		new(time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)))
	teamADriver := testhelpers.SeedTeamDriver(
		t,
		teamA.ID,
		driver.ID,
		driverJoinedTeamA,
		&driverLeftTeamA,
	)
	teamBDriver := testhelpers.SeedTeamDriver(
		t,
		teamB.ID,
		driver.ID,
		driverJoinedTeamB,
		&driverLeftTeamB,
	)
	teamCDriver := testhelpers.SeedTeamDriver(t, teamC.ID, driver.ID, driverJoinedTeamC, nil)

	tests := []struct {
		name     string // description of this test case
		seasonID int32
		driverID int32
		when     time.Time
		want     *models.TeamDriver
		wantErr  bool
	}{
		{
			name: "lower edge teamA range", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), want: teamADriver,
		},
		{ // note: leftAt is not inclusive, so no team association at this time
			name: "upper edge teamA range", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), want: nil, wantErr: true,
		},
		{
			name: "in teamA range", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), want: teamADriver,
		},
		{
			name: "in teamB range", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), want: teamBDriver,
		},
		{ // note: leftAtB equal to joinedAtC -> will teamC is it
			name:     "upper edge teamB equal to lower edge teamC",
			seasonID: season.ID,
			driverID: driver.ID,
			when:     time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
			want:     teamCDriver,
		},
		{
			name: "in teamC range", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC), want: teamCDriver,
		},
		{
			name: "after teamC has left", seasonID: season.ID, driverID: driver.ID,
			when: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), want: nil, wantErr: true,
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.

			got, gotErr := repo.QueryTeamDrivers().
				ResolveTeamDriver(ctx, tt.seasonID, tt.driverID, tt.when)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ResolveTeamDriver() failed: %v", gotErr)
				}
				if !errors.Is(gotErr, repoerrors.ErrNotFound) {
					t.Errorf("ResolveTeamDriver() failed with unexpected error: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ResolveTeamDriver() succeeded unexpectedly")
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveTeamDriver() = %v, want %v", got, tt.want)
			}
		})
	}
}
