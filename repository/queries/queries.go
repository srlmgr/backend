//nolint:lll //readability
package queries

import (
	"github.com/jackc/pgx/v5/pgxpool"

	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/pgbob"
)

type (
	queries struct {
		resolver       rootrepo.Resolver
		qSeasonDrivers rootrepo.QuerySeasonDriver
		qTeamDrivers   rootrepo.QueryTeamDriver
		qCarClasses    rootrepo.QueryCarClass
		qTrackLayouts  rootrepo.QueryTrackLayouts
	}
)

var _ rootrepo.Queries = (*queries)(nil)

// New returns a postgres-backed QueryRepository.
func New(pool *pgxpool.Pool) rootrepo.Queries {
	exec := pgbob.New(pool)
	return &queries{
		resolver:       NewResolver(exec),
		qSeasonDrivers: NewSeasonDriverQueries(exec),
		qTeamDrivers:   NewTeamDriverQueries(exec),
		qCarClasses:    NewCarClassQueries(exec),
		qTrackLayouts:  NewTrackLayoutQueries(exec),
	}
}

func (r *queries) QueryResolver() rootrepo.Resolver { return r.resolver }

func (r *queries) QuerySeasonDrivers() rootrepo.QuerySeasonDriver { return r.qSeasonDrivers }
func (r *queries) QueryTeamDrivers() rootrepo.QueryTeamDriver     { return r.qTeamDrivers }
func (r *queries) QueryCarClasses() rootrepo.QueryCarClass        { return r.qCarClasses }
func (r *queries) QueryTrackLayouts() rootrepo.QueryTrackLayouts  { return r.qTrackLayouts }
