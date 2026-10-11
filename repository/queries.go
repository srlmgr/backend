// contains queries for repositories that do not fit into a specific repository,
// such as queries that span multiple repositories or are used by multiple repositories.
package repository

import (
	"context"
	"time"

	"github.com/srlmgr/backend/db/models"
)

type (
	TrackLayoutContainer struct {
		TrackLayout *models.TrackLayout `json:"trackLayout"`
		Track       *models.Track       `json:"track"`
	}

	SeasonTarget struct {
		EventID    int32
		RaceID     int32
		RaceGridID int32
	}
)

type (
	Resolver interface {
		ResolveEvent(
			ctx context.Context,
			seasonID, eventNo int32,
		) (*SeasonTarget, error)
		ResolveRace(
			ctx context.Context,
			seasonID, eventNo, raceNo int32,
		) (*SeasonTarget, error)
		ResolveRaceGrid(
			ctx context.Context,
			seasonID, eventNo, raceNo, raceGridNo int32,
		) (*SeasonTarget, error)
	}
	QuerySeasonDriver interface {
		ResolveSeasonDriver(
			ctx context.Context,
			seasonID, driverID int32,
			when time.Time) (
			*models.SeasonDriver, error,
		)
	}
	QueryTeamDriver interface {
		FindBySeasonAndDriver(ctx context.Context, seasonID, driverID int32) (
			*models.TeamDriver, error,
		)
		ResolveTeamDriver(
			ctx context.Context,
			seasonID, driverID int32,
			when time.Time) (
			*models.TeamDriver, error,
		)
		FindBySeason(ctx context.Context, seasonID int32) (
			[]*models.TeamDriver, error,
		)
	}

	QueryCarClass interface {
		FindBySeasonAndCarModelVariant(
			ctx context.Context,
			seasonID, carModelVariantID int32) (
			*models.CarClass, error,
		)
	}
	QueryTrackLayouts interface {
		GetAll(ctx context.Context) ([]*TrackLayoutContainer, error)
		ForSimulationID(ctx context.Context, simulationID int32) (
			[]*TrackLayoutContainer, error,
		)
	}
	Queries interface {
		QueryResolver() Resolver
		QuerySeasonDrivers() QuerySeasonDriver
		QueryTeamDrivers() QueryTeamDriver
		QueryCarClasses() QueryCarClass
		QueryTrackLayouts() QueryTrackLayouts
	}
)
