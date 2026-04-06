// contains queries for repositories that do not fit into a specific repository,
// such as queries that span multiple repositories or are used by multiple repositories.
package repository

import (
	"context"

	"github.com/srlmgr/backend/db/models"
)

type (
	QueryTeamDriver interface {
		FindBySeasonAndDriver(ctx context.Context, seasonID, driverID int32) (
			*models.TeamDriver, error,
		)
	}
	Queries interface {
		QueryTeamDrivers() QueryTeamDriver
	}
)
