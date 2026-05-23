// contains queries for repositories that do not fit into a specific repository,
// such as queries that span multiple repositories or are used by multiple repositories.
package repository

import (
	"context"

	"github.com/srlmgr/backend/db/models"
)

type (
	TrackLayoutContainer struct {
		TrackLayout *models.TrackLayout `json:"trackLayout"`
		Track       *models.Track       `json:"track"`
	}
)

type (
	QueryTeamDriver interface {
		FindBySeasonAndDriver(ctx context.Context, seasonID, driverID int32) (
			*models.TeamDriver, error,
		)
		FindBySeason(ctx context.Context, seasonID int32) (
			[]*models.TeamDriver, error,
		)
	}
	QueryCarClass interface {
		FindBySeasonAndCarModel(ctx context.Context, seasonID, carModelID int32) (
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
		QueryTeamDrivers() QueryTeamDriver
		QueryCarClasses() QueryCarClass
		QueryTrackLayouts() QueryTrackLayouts
	}
)
