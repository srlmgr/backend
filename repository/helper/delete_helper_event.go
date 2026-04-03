//nolint:dupl // delete queries follow the same pattern per entity
package helper

import (
	"context"
	"fmt"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/pgbob"
)

var ErrNoBobExecutorInContext = fmt.Errorf("no bob.Executor found in context")

// deletes all data related to an event, such as
// races, grids, results, points, etc. Does not delete the event itself.
// ctx MUST provide a bob.Executor
//
//nolint:whitespace //editor/linter issue
func DeleteEventRelated(
	ctx context.Context,
	eventID int32,
) error {
	if err := DeleteEventBookingEntries(ctx, eventID); err != nil {
		return err
	}
	if err := DeleteEventResultEntries(ctx, eventID); err != nil {
		return err
	}
	if err := DeleteEventImportBatches(ctx, eventID); err != nil {
		return err
	}
	if err := DeleteEventRaceGrids(ctx, eventID); err != nil {
		return err
	}
	if err := DeleteEventRaces(ctx, eventID); err != nil {
		return err
	}
	return nil
}

//nolint:whitespace //editor/linter issue
func DeleteEventRaceGrids(
	ctx context.Context,
	eventID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}
	subQuery := psql.Select(
		sm.Columns(models.RaceGrids.Columns.ID),
		sm.From(models.RaceGrids.Name()),
		models.SelectJoins.RaceGrids.InnerJoin.Race,
		models.SelectWhere.Races.EventID.EQ(eventID),
	)
	_, err := models.RaceGrids.Delete(
		dm.Where(models.RaceGrids.Columns.ID.In(subQuery)),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteEventBookingEntries(
	ctx context.Context,
	eventID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.BookingEntries.Delete(
		dm.Where(models.BookingEntries.Columns.EventID.EQ(psql.Arg(eventID))),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteEventRaces(
	ctx context.Context,
	eventID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.Races.Delete(
		dm.Where(models.Races.Columns.EventID.EQ(psql.Arg(eventID))),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteEventResultEntries(
	ctx context.Context,
	eventID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}
	subQuery := psql.Select(
		sm.Columns(models.Races.Columns.ID),
		sm.From(models.Races.Name()),
		models.SelectWhere.Races.EventID.EQ(eventID),
	)
	_, err := models.ResultEntries.Delete(
		dm.Where(models.ResultEntries.Columns.RaceID.In(subQuery)),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteEventImportBatches(
	ctx context.Context,
	eventID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}
	subQuery := psql.Select(
		sm.Columns(models.Races.Columns.ID),
		sm.From(models.Races.Name()),
		models.SelectWhere.Races.EventID.EQ(eventID),
	)
	_, err := models.ImportBatches.Delete(
		dm.Where(models.ImportBatches.Columns.RaceID.In(subQuery)),
	).Exec(ctx, executor)
	return err
}
