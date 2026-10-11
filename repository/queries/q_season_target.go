package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/repoerrors"
)

func NewResolver(exec *pgbob.Executor) repository.Resolver {
	return &queryResolver{exec: exec}
}

type queryResolver struct {
	exec *pgbob.Executor
}

type qSeasonTargetData struct {
	EventID    int32 `db:"event_id"`
	RaceID     int32 `db:"race_id"`
	RaceGridID int32 `db:"race_grid_id"`
}

// ResolveEvent resolves the event sequence number to the event ID.
//
//nolint:whitespace // editor/linter issue
func (r *queryResolver) ResolveEvent(
	ctx context.Context,
	seasonID, eventNo int32,
) (*repository.SeasonTarget, error) {
	e := models.Events.Columns.AliasedAs("e")

	row, err := bob.One(ctx, r.getExecutor(ctx),
		psql.Select(
			sm.Columns(e.ID.As("event_id")),
			sm.From(models.Events.NameExpr().As(e.Alias())),
			sm.Where(e.SeasonID.EQ(psql.Arg(seasonID))),
			sm.Where(e.SequenceNo.EQ(psql.Arg(eventNo))),
		),
		scan.StructMapper[qSeasonTargetData](),
	)
	if err != nil {
		return nil, notFoundOr(err, "event %d in season %d", eventNo, seasonID)
	}

	return &repository.SeasonTarget{EventID: row.EventID}, nil
}

// ResolveRace resolves the event and race sequence numbers to the event and race ID.
//
//nolint:whitespace // editor/linter issue
func (r *queryResolver) ResolveRace(
	ctx context.Context,
	seasonID, eventNo, raceNo int32,
) (*repository.SeasonTarget, error) {
	e := models.Events.Columns.AliasedAs("e")
	rc := models.Races.Columns.AliasedAs("rc")

	row, err := bob.One(ctx, r.getExecutor(ctx),
		psql.Select(
			sm.Columns(e.ID.As("event_id"), rc.ID.As("race_id")),
			sm.From(models.Events.NameExpr().As(e.Alias())),
			sm.InnerJoin(models.Races.NameExpr().As(rc.Alias())).
				On(rc.EventID.EQ(e.ID)),
			sm.Where(e.SeasonID.EQ(psql.Arg(seasonID))),
			sm.Where(e.SequenceNo.EQ(psql.Arg(eventNo))),
			sm.Where(rc.SequenceNo.EQ(psql.Arg(raceNo))),
		),
		scan.StructMapper[qSeasonTargetData](),
	)
	if err != nil {
		return nil, notFoundOr(err,
			"race %d in event %d of season %d", raceNo, eventNo, seasonID)
	}

	return &repository.SeasonTarget{EventID: row.EventID, RaceID: row.RaceID}, nil
}

// ResolveRaceGrid resolves the event, race and race grid sequence numbers to the
// event, race and race grid ID.
//
//nolint:whitespace // editor/linter issue
func (r *queryResolver) ResolveRaceGrid(
	ctx context.Context,
	seasonID, eventNo, raceNo, raceGridNo int32,
) (*repository.SeasonTarget, error) {
	e := models.Events.Columns.AliasedAs("e")
	rc := models.Races.Columns.AliasedAs("rc")
	rg := models.RaceGrids.Columns.AliasedAs("rg")

	row, err := bob.One(ctx, r.getExecutor(ctx),
		psql.Select(
			sm.Columns(
				e.ID.As("event_id"),
				rc.ID.As("race_id"),
				rg.ID.As("race_grid_id"),
			),
			sm.From(models.Events.NameExpr().As(e.Alias())),
			sm.InnerJoin(models.Races.NameExpr().As(rc.Alias())).
				On(rc.EventID.EQ(e.ID)),
			sm.InnerJoin(models.RaceGrids.NameExpr().As(rg.Alias())).
				On(rg.RaceID.EQ(rc.ID)),
			sm.Where(e.SeasonID.EQ(psql.Arg(seasonID))),
			sm.Where(e.SequenceNo.EQ(psql.Arg(eventNo))),
			sm.Where(rc.SequenceNo.EQ(psql.Arg(raceNo))),
			sm.Where(rg.SequenceNo.EQ(psql.Arg(raceGridNo))),
		),
		scan.StructMapper[qSeasonTargetData](),
	)
	if err != nil {
		return nil, notFoundOr(err,
			"race grid %d in race %d of event %d in season %d",
			raceGridNo, raceNo, eventNo, seasonID)
	}

	return &repository.SeasonTarget{
		EventID:    row.EventID,
		RaceID:     row.RaceID,
		RaceGridID: row.RaceGridID,
	}, nil
}

// notFoundOr maps sql.ErrNoRows to repoerrors.ErrNotFound, other errors are kept.
func notFoundOr(err error, format string, args ...any) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), repoerrors.ErrNotFound)
	}
	return err
}

func (r *queryResolver) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	return r.exec
}
