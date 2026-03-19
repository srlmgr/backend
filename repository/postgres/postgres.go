// Package postgres wires the repository interfaces to postgres-backed implementations.
//
//nolint:lll // readability
package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"

	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/bookingentries"
	"github.com/srlmgr/backend/repository/cars"
	"github.com/srlmgr/backend/repository/drivers"
	"github.com/srlmgr/backend/repository/eventprocessingaudit"
	"github.com/srlmgr/backend/repository/events"
	"github.com/srlmgr/backend/repository/importbatches"
	"github.com/srlmgr/backend/repository/pointsystems"
	"github.com/srlmgr/backend/repository/races"
	"github.com/srlmgr/backend/repository/racingsims"
	"github.com/srlmgr/backend/repository/resultentries"
	"github.com/srlmgr/backend/repository/seasons"
	"github.com/srlmgr/backend/repository/series"
	"github.com/srlmgr/backend/repository/standings"
	"github.com/srlmgr/backend/repository/teams"
	"github.com/srlmgr/backend/repository/tracks"
)

type repository struct {
	racingSims           racingsims.Repository
	pointSystems         pointsystems.Repository
	drivers              drivers.Repository
	tracks               tracks.Repository
	cars                 cars.Repository
	series               series.Repository
	seasons              seasons.Repository
	events               events.Repository
	races                races.Repository
	teams                teams.Repository
	importBatches        importbatches.Repository
	resultEntries        resultentries.Repository
	bookingEntries       bookingentries.Repository
	eventProcessingAudit eventprocessingaudit.Repository
	standings            standings.Repository
}

// New returns the root postgres-backed repository aggregate.
func New(pool *pgxpool.Pool) rootrepo.Repository {
	return &repository{
		racingSims:           racingsims.New(pool),
		pointSystems:         pointsystems.New(pool),
		drivers:              drivers.New(pool),
		tracks:               tracks.New(pool),
		cars:                 cars.New(pool),
		series:               series.New(pool),
		seasons:              seasons.New(pool),
		events:               events.New(pool),
		races:                races.New(pool),
		teams:                teams.New(pool),
		importBatches:        importbatches.New(pool),
		resultEntries:        resultentries.New(pool),
		bookingEntries:       bookingentries.New(pool),
		eventProcessingAudit: eventprocessingaudit.New(pool),
		standings:            standings.New(pool),
	}
}

func (r *repository) RacingSims() racingsims.Repository         { return r.racingSims }
func (r *repository) PointSystems() pointsystems.Repository     { return r.pointSystems }
func (r *repository) Drivers() drivers.Repository               { return r.drivers }
func (r *repository) Tracks() tracks.Repository                 { return r.tracks }
func (r *repository) Cars() cars.Repository                     { return r.cars }
func (r *repository) Series() series.Repository                 { return r.series }
func (r *repository) Seasons() seasons.Repository               { return r.seasons }
func (r *repository) Events() events.Repository                 { return r.events }
func (r *repository) Races() races.Repository                   { return r.races }
func (r *repository) Teams() teams.Repository                   { return r.teams }
func (r *repository) ImportBatches() importbatches.Repository   { return r.importBatches }
func (r *repository) ResultEntries() resultentries.Repository   { return r.resultEntries }
func (r *repository) BookingEntries() bookingentries.Repository { return r.bookingEntries }
func (r *repository) EventProcessingAudit() eventprocessingaudit.Repository {
	return r.eventProcessingAudit
}
func (r *repository) Standings() standings.Repository { return r.standings }
