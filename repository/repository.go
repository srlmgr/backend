// Package repository defines the persistence interfaces for all entity groups.
package repository

import (
	"github.com/srlmgr/backend/repository/bookingentries"
	"github.com/srlmgr/backend/repository/cars"
	"github.com/srlmgr/backend/repository/drivers"
	"github.com/srlmgr/backend/repository/eventprocessingaudit"
	"github.com/srlmgr/backend/repository/events"
	"github.com/srlmgr/backend/repository/importbatches"
	"github.com/srlmgr/backend/repository/pointsystems"
	"github.com/srlmgr/backend/repository/races"
	"github.com/srlmgr/backend/repository/racingsims"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/repository/resultentries"
	"github.com/srlmgr/backend/repository/seasons"
	"github.com/srlmgr/backend/repository/series"
	"github.com/srlmgr/backend/repository/standings"
	"github.com/srlmgr/backend/repository/teams"
	"github.com/srlmgr/backend/repository/tracks"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = repoerrors.ErrNotFound

// Repository collects all entity-group repositories.
type Repository interface {
	RacingSims() racingsims.Repository
	PointSystems() pointsystems.Repository
	Drivers() drivers.Repository
	Tracks() tracks.Repository
	Cars() cars.Repository
	Series() series.Repository
	Seasons() seasons.Repository
	Events() events.Repository
	Races() races.Repository
	Teams() teams.Repository
	ImportBatches() importbatches.Repository
	ResultEntries() resultentries.Repository
	BookingEntries() bookingentries.Repository
	EventProcessingAudit() eventprocessingaudit.Repository
	Standings() standings.Repository
}
