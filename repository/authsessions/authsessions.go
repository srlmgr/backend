// Package authsessions provides persistence for authentication sessions.
//
//nolint:lll,whitespace // repository signatures and query expressions
package authsessions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/scan"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// Repository defines persistence operations for authentication sessions.
type Repository interface {
	Check(ctx context.Context) error
	Create(ctx context.Context, input *models.AuthSessionSetter) (*models.AuthSession, error)
	LoadByHash(ctx context.Context, hash []byte, forUpdate bool) (*models.AuthSession, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.AuthSessionSetter,
	) (*models.AuthSession, error)
	DeleteByHash(ctx context.Context, hash []byte) error
	CleanupExpired(ctx context.Context, limit int) (int64, error)
}

type authSessionsRepository struct{ exec *pgbob.Executor }

// New returns a PostgreSQL-backed repository.
func New(pool *pgxpool.Pool) Repository {
	return &authSessionsRepository{exec: pgbob.New(pool)}
}

type permissions struct {
	Select   bool `db:"can_select"`
	Insert   bool `db:"can_insert"`
	Update   bool `db:"can_update"`
	Delete   bool `db:"can_delete"`
	Sequence bool `db:"can_sequence"`
}

func (r *authSessionsRepository) Check(ctx context.Context) error {
	if _, err := models.AuthSessions.Query(sm.Limit(0)).All(ctx, r.getExecutor(ctx)); err != nil {
		return fmt.Errorf("check auth_sessions table: %w", err)
	}
	table := psql.Arg("auth_sessions")
	result, err := bob.One(ctx, r.getExecutor(ctx), psql.Select(
		sm.Columns(
			psql.Group(psql.F("has_table_privilege", table, psql.Arg("SELECT"))).As("can_select"),
			psql.Group(psql.F("has_table_privilege", table, psql.Arg("INSERT"))).As("can_insert"),
			psql.Group(psql.F("has_table_privilege", table, psql.Arg("UPDATE"))).As("can_update"),
			psql.Group(psql.F("has_table_privilege", table, psql.Arg("DELETE"))).As("can_delete"),
			psql.Group(psql.F("has_sequence_privilege",
				psql.F("pg_get_serial_sequence", table, psql.Arg("id")),
				psql.Arg("USAGE"))).As("can_sequence"),
		),
	), scan.StructMapper[permissions]())
	if err != nil {
		return fmt.Errorf("check auth_sessions permissions: %w", err)
	}
	if !result.Select || !result.Insert || !result.Update || !result.Delete || !result.Sequence {
		return errors.New("insufficient auth_sessions table permissions")
	}
	return nil
}

func (r *authSessionsRepository) Create(
	ctx context.Context, input *models.AuthSessionSetter,
) (*models.AuthSession, error) {
	return models.AuthSessions.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *authSessionsRepository) LoadByHash(
	ctx context.Context, hash []byte, forUpdate bool,
) (*models.AuthSession, error) {
	if forUpdate && pgbob.FromContext(ctx) == nil {
		return nil, errors.New("locking a session requires a transaction")
	}
	mods := []bob.Mod[*dialect.SelectQuery]{
		sm.Where(models.AuthSessions.Columns.SessionIDHash.EQ(psql.Arg(hash))),
		sm.Where(models.AuthSessions.Columns.ExpiresAt.IsNull().
			Or(models.AuthSessions.Columns.ExpiresAt.GT(psql.F("clock_timestamp")))),
	}
	if forUpdate {
		mods = append(mods, sm.ForUpdate())
	}
	session, err := models.AuthSessions.Query(mods...).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerrors.ErrNotFound
	}
	return session, err
}

func (r *authSessionsRepository) Update(
	ctx context.Context, id int32, input *models.AuthSessionSetter,
) (*models.AuthSession, error) {
	session, err := models.AuthSessions.Update(input.UpdateMod(),
		um.Where(models.AuthSessions.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerrors.ErrNotFound
	}
	return session, err
}

func (r *authSessionsRepository) DeleteByHash(ctx context.Context, hash []byte) error {
	_, err := models.AuthSessions.Delete(
		dm.Where(models.AuthSessions.Columns.SessionIDHash.EQ(psql.Arg(hash))),
	).Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *authSessionsRepository) CleanupExpired(ctx context.Context, limit int) (int64, error) {
	if limit <= 0 {
		return 0, errors.New("cleanup limit must be positive")
	}
	expired := psql.Select(
		sm.Columns(models.AuthSessions.Columns.ID),
		sm.From(models.AuthSessions.Name()),
		sm.Where(models.AuthSessions.Columns.ExpiresAt.LTE(psql.F("now"))),
		sm.OrderBy(models.AuthSessions.Columns.ExpiresAt),
		sm.Limit(psql.Arg(limit)),
		sm.ForUpdate().SkipLocked(),
	)
	return models.AuthSessions.Delete(
		dm.Where(models.AuthSessions.Columns.ID.In(expired)),
	).Exec(ctx, r.getExecutor(ctx))
}

func (r *authSessionsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
