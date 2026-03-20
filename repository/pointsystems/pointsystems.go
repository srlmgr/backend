// Package pointsystems provides repositories for the point_systems migration group.
//
//nolint:lll,whitespace // repository implementations can be verbose
package pointsystems

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

// PointSystemsRepository defines persistence operations for PointSystem entities.
type PointSystemsRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.PointSystem, error)
	LoadAll(ctx context.Context) ([]*models.PointSystem, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.PointSystemSetter) (*models.PointSystem, error)
	Update(
		ctx context.Context,
		id int32,
		input *models.PointSystemSetter,
	) (*models.PointSystem, error)
}

// PointRulesRepository defines persistence operations for PointRule entities.
type PointRulesRepository interface {
	LoadByID(ctx context.Context, id int32) (*models.PointRule, error)
	DeleteByID(ctx context.Context, id int32) error
	Create(ctx context.Context, input *models.PointRuleSetter) (*models.PointRule, error)
	Update(ctx context.Context, id int32, input *models.PointRuleSetter) (*models.PointRule, error)
}

// Repository exposes repositories for the point_systems migration group.
type Repository interface {
	PointSystems() PointSystemsRepository
	PointRules() PointRulesRepository
}

type repository struct {
	pointSystems PointSystemsRepository
	pointRules   PointRulesRepository
}

type (
	pointSystemsRepository struct{ exec *pgbob.Executor }
	pointRulesRepository   struct{ exec *pgbob.Executor }
)

// New returns a postgres-backed Repository.
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		pointSystems: &pointSystemsRepository{exec: pgbob.New(pool)},
		pointRules:   &pointRulesRepository{exec: pgbob.New(pool)},
	}
}

func (r *repository) PointSystems() PointSystemsRepository { return r.pointSystems }
func (r *repository) PointRules() PointRulesRepository     { return r.pointRules }

func (r *pointSystemsRepository) LoadByID(
	ctx context.Context,
	id int32,
) (*models.PointSystem, error) {
	entity, err := models.PointSystems.Query(sm.Where(models.PointSystems.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("point system %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *pointSystemsRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.PointSystems.Delete(dm.Where(models.PointSystems.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *pointSystemsRepository) Create(
	ctx context.Context,
	input *models.PointSystemSetter,
) (*models.PointSystem, error) {
	return models.PointSystems.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *pointSystemsRepository) LoadAll(ctx context.Context) ([]*models.PointSystem, error) {
	return models.PointSystems.Query().All(ctx, r.getExecutor(ctx))
}

func (r *pointSystemsRepository) Update(
	ctx context.Context,
	id int32,
	input *models.PointSystemSetter,
) (*models.PointSystem, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *pointRulesRepository) LoadByID(ctx context.Context, id int32) (*models.PointRule, error) {
	entity, err := models.PointRules.Query(sm.Where(models.PointRules.Columns.ID.EQ(psql.Arg(id)))).
		One(ctx, r.getExecutor(ctx))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("point rule %d: %w", id, repoerrors.ErrNotFound)
	}
	return entity, err
}

func (r *pointRulesRepository) DeleteByID(ctx context.Context, id int32) error {
	_, err := models.PointRules.Delete(dm.Where(models.PointRules.Columns.ID.EQ(psql.Arg(id)))).
		Exec(ctx, r.getExecutor(ctx))
	return err
}

func (r *pointRulesRepository) Create(
	ctx context.Context,
	input *models.PointRuleSetter,
) (*models.PointRule, error) {
	return models.PointRules.Insert(input).One(ctx, r.getExecutor(ctx))
}

func (r *pointRulesRepository) Update(
	ctx context.Context,
	id int32,
	input *models.PointRuleSetter,
) (*models.PointRule, error) {
	entity, err := r.LoadByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entity.Update(ctx, r.getExecutor(ctx), input); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *pointSystemsRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}

func (r *pointRulesRepository) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
