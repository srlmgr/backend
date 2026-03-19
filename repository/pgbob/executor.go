// Package pgbob adapts a pgx pool to the executor interface expected by Bob.
package pgbob

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

// Executor adapts *sql.DB to Bob's executor interface by exposing scan.Rows.
type Executor struct {
	db *sql.DB
}

// New creates an executor backed by the provided pgx pool.
func New(pool *pgxpool.Pool) *Executor {
	return &Executor{db: stdlib.OpenDBFromPool(pool)}
}

// QueryContext executes a query and returns scan.Rows for Bob.
//
//nolint:whitespace //editor/linter issue
func (e *Executor) QueryContext(ctx context.Context, query string, args ...any) (
	scan.Rows, error,
) {
	return e.db.QueryContext(ctx, query, args...)
}

// ExecContext executes a statement and returns the standard sql.Result.
//
//nolint:whitespace //editor/linter issue
func (e *Executor) ExecContext(ctx context.Context, query string, args ...any) (
	sql.Result, error,
) {
	return e.db.ExecContext(ctx, query, args...)
}

type bopContextKey struct{}

func NewContext(ctx context.Context, executor bob.Executor) context.Context {
	return context.WithValue(ctx, bopContextKey{}, executor)
}

func FromContext(ctx context.Context) bob.Executor {
	if ctx == nil {
		return nil
	}
	if executor, ok := ctx.Value(bopContextKey{}).(bob.Executor); ok {
		return executor
	}
	return nil
}
