// Package repository defines the persistence interfaces for all entity groups.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob"

	"github.com/srlmgr/backend/repository/bookingentries"
	"github.com/srlmgr/backend/repository/cars"
	"github.com/srlmgr/backend/repository/drivers"
	"github.com/srlmgr/backend/repository/eventprocessingaudit"
	"github.com/srlmgr/backend/repository/events"
	"github.com/srlmgr/backend/repository/importbatches"
	"github.com/srlmgr/backend/repository/pgbob"
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

type TransactionManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type bobTransaction struct {
	db *bob.DB
}

var _ TransactionManager = (*bobTransaction)(nil)

func NewBobTransactionFromPool(pool *pgxpool.Pool) TransactionManager {
	x := bob.NewDB(stdlib.OpenDBFromPool(pool))
	return &bobTransaction{
		db: &x,
	}
}

// the contract with the repositories is:
// we put the current executor into the context, the repository should first look
// in the context for an executor and then use it to execute queries
//
//nolint:whitespace //editor/linter issue
func (b *bobTransaction) RunInTx(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	return b.db.RunInTx(ctx, nil, func(ctx context.Context, e bob.Executor) error {
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = pgbob.NewContext(ctx, e)
		return fn(ctx)
	})
}
