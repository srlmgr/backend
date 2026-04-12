// Package repository defines the persistence interfaces for all entity groups.
package repository

import (
	"context"
	"errors"
	"fmt"

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
	Queries() Queries
}

type TransactionManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type bobTransaction struct {
	db *bob.DB
}

var _ TransactionManager = (*bobTransaction)(nil)

type repositoryKey struct{} // used type to store repository in context
var repositoryKeyInstance = repositoryKey{}

func AddToContext(ctx context.Context, r Repository) context.Context {
	return context.WithValue(ctx, repositoryKeyInstance, r)
}

func GetFromContext(ctx context.Context) Repository {
	if ctx == nil {
		return nil
	}
	if r, ok := ctx.Value(repositoryKeyInstance).(Repository); ok {
		return r
	}
	return nil
}

func NewBobTransactionFromPool(pool *pgxpool.Pool) TransactionManager {
	x := bob.NewDB(stdlib.OpenDBFromPool(pool))
	return &bobTransaction{
		db: &x,
	}
}

// the contract with the repositories is:
// we put the current executor into the context, the repository should first look
// in the context for an executor and then use it to execute queries
// the code is borrowd from bob's RunInTx implementation,
// but here we return the original error of the executing function
//
//nolint:whitespace //editor/linter issue
func (b *bobTransaction) RunInTx(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = pgbob.NewContext(ctx, tx)
	// if an error occurs we want to return the original bob error
	// because we may want to check for specific error types
	// (e.g. unique constraint violation) in the service layer
	if err := fn(ctx); err != nil {
		txErr := fmt.Errorf("call: %w", err)

		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return errors.Join(txErr, rollbackErr)
		}

		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

//nolint:whitespace //keep it as a reference
func (b *bobTransaction) RunInTxOld(
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
