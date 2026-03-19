// Package eventprocessingaudit provides repositories for the event_processing_audit migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package eventprocessingaudit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// Repository defines persistence operations for EventProcessingAudit entities.
type Repository interface {
	LoadByID(ctx context.Context, id int32) (*models.EventProcessingAudit, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(
		ctx context.Context,
		input *models.EventProcessingAuditSetter,
	) (*models.EventProcessingAudit, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.EventProcessingAuditSetter,
	) (*models.EventProcessingAudit, error)
}

type auditRepository struct{ exec *pgbob.Executor }

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &auditRepository{exec: pgbob.New(pool)}
}

func (r *auditRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.EventProcessingAudit, error) {
	entity, err := models.EventProcessingAudits.Query(sm.Where(models.EventProcessingAudits.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("event processing audit %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *auditRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.EventProcessingAudits.Delete(dm.Where(models.EventProcessingAudits.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *auditRepository) Create(
	ctx context.Context,
	input *models.EventProcessingAuditSetter,
) (*models.EventProcessingAudit, error) {
	return models.EventProcessingAudits.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *auditRepository) Update(
	ctx context.Context,
	id int32,
	input *models.EventProcessingAuditSetter,
) (*models.EventProcessingAudit, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *auditRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
