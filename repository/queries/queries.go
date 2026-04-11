package queries

import (
	"github.com/jackc/pgx/v5/pgxpool"

	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/pgbob"
)

type (
	queries struct {
		qTeamDrivers rootrepo.QueryTeamDriver
		qCarClasses  rootrepo.QueryCarClass
	}
)

var _ rootrepo.Queries = (*queries)(nil)

// New returns a postgres-backed QueryRepository.
func New(pool *pgxpool.Pool) rootrepo.Queries {
	return &queries{
		qTeamDrivers: NewTeamDriverQueries(pgbob.New(pool)),
		qCarClasses:  NewCarClassQueries(pgbob.New(pool)),
	}
}
func (r *queries) QueryTeamDrivers() rootrepo.QueryTeamDriver { return r.qTeamDrivers }
func (r *queries) QueryCarClasses() rootrepo.QueryCarClass    { return r.qCarClasses }
