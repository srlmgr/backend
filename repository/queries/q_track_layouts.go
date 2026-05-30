package queries

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/pgbob"
)

func NewTrackLayoutQueries(exec *pgbob.Executor) repository.QueryTrackLayouts {
	return &queryTrackLayouts{exec: exec}
}

type queryTrackLayouts struct {
	exec *pgbob.Executor
}
type qTLData struct {
	TrackLayout models.TrackLayout `db:",inline"`
	Track       models.Track       `db:",inline"`
}

//nolint:whitespace // editor/linter issue
func (r *queryTrackLayouts) GetAll(
	ctx context.Context,
) ([]*repository.TrackLayoutContainer, error) {
	tl := models.TrackLayouts.Columns.AliasedAs("tl")
	t := models.Tracks.Columns.AliasedAs("t")

	rows, err := bob.All(ctx, r.getExecutor(ctx),
		psql.Select(
			sm.Columns(tl.WithPrefix("track_layout."), t.WithPrefix("track.")),
			sm.From(models.TrackLayouts.NameExpr().As(tl.Alias())),
			sm.InnerJoin(models.Tracks.NameExpr().As(t.Alias())).On(t.ID.EQ(tl.TrackID)),
		),
		scan.StructMapper[qTLData](),
	)
	if err != nil {
		return nil, err
	}

	containers := make([]*repository.TrackLayoutContainer, 0, len(rows))
	for i := range rows {
		dbRow := &rows[i]
		containers = append(containers, &repository.TrackLayoutContainer{
			TrackLayout: &dbRow.TrackLayout,
			Track:       &dbRow.Track,
		})
	}

	return containers, nil
}

//nolint:whitespace // readability, editor/linter issue
func (r *queryTrackLayouts) ForSimulationID(
	ctx context.Context,
	simulationID int32,
) ([]*repository.TrackLayoutContainer, error) {
	stla := models.SimulationTrackLayoutAliases.Columns.AliasedAs("stla")
	tl := models.TrackLayouts.Columns.AliasedAs("tl")
	t := models.Tracks.Columns.AliasedAs("t")

	rows, err := bob.All(ctx, r.getExecutor(ctx),
		psql.Select(
			sm.Columns(tl.WithPrefix("track_layout."), t.WithPrefix("track.")),
			sm.From(models.SimulationTrackLayoutAliases.NameExpr().As(stla.Alias())),
			sm.InnerJoin(models.TrackLayouts.NameExpr().As(tl.Alias())).
				On(tl.ID.EQ(stla.TrackLayoutID)),
			sm.InnerJoin(models.Tracks.NameExpr().As(t.Alias())).On(t.ID.EQ(tl.TrackID)),
			sm.Where(stla.SimulationID.EQ(psql.Arg(simulationID))),
		),
		scan.StructMapper[qTLData](),
	)
	if err != nil {
		return nil, err
	}

	containers := make([]*repository.TrackLayoutContainer, 0, len(rows))
	for i := range rows {
		dbRow := &rows[i]
		containers = append(containers, &repository.TrackLayoutContainer{
			TrackLayout: &dbRow.TrackLayout,
			Track:       &dbRow.Track,
		})
	}

	return containers, nil
}

func (r *queryTrackLayouts) getExecutor(ctx context.Context) bob.Executor {
	if executor := pgbob.FromContext(ctx); executor != nil {
		return executor
	}
	// use bob.Debug(r.exec) to log queries if no executor is found in the context
	return r.exec
}
