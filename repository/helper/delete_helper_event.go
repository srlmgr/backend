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
	if err := DeleteEventDriverStandings(ctx, eventID); err != nil {
		return err
	}
	if err := DeleteEventTeamStandings(ctx, eventID); err != nil {
		return err
	}
	if err := DeleteEventProcessingAudit(ctx, eventID); err != nil {
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
		sm.Columns(models.RaceGrids.Columns.ID),
		sm.From(models.RaceGrids.Name()),
		models.SelectJoins.RaceGrids.InnerJoin.Race,
		models.SelectWhere.Races.EventID.EQ(eventID),
	)

	_, err := models.ResultEntries.Delete(
		dm.Where(models.ResultEntries.Columns.RaceGridID.In(subQuery)),
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
		sm.Columns(models.RaceGrids.Columns.ID),
		sm.From(models.RaceGrids.Name()),
		models.SelectJoins.RaceGrids.InnerJoin.Race,
		models.SelectWhere.Races.EventID.EQ(eventID),
	)
	_, err := models.ImportBatches.Delete(
		dm.Where(models.ImportBatches.Columns.RaceGridID.In(subQuery)),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteEventDriverStandings(
	ctx context.Context,
	eventID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.EventDriverStandings.Delete(
		dm.Where(models.EventDriverStandings.Columns.EventID.EQ(psql.Arg(eventID))),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteEventTeamStandings(
	ctx context.Context,
	eventID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.EventTeamStandings.Delete(
		dm.Where(models.EventTeamStandings.Columns.EventID.EQ(psql.Arg(eventID))),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteEventProcessingAudit(
	ctx context.Context,
	eventID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.EventProcessingAudits.Delete(
		dm.Where(models.EventProcessingAudits.Columns.EventID.EQ(psql.Arg(eventID))),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteRaceRelated(
	ctx context.Context,
	raceID int32,
) error {
	if err := DeleteRaceBookingEntries(ctx, raceID); err != nil {
		return err
	}
	if err := DeleteRaceResultEntries(ctx, raceID); err != nil {
		return err
	}
	if err := DeleteRaceImportBatches(ctx, raceID); err != nil {
		return err
	}
	if err := DeleteRaceRaceGrids(ctx, raceID); err != nil {
		return err
	}
	return nil
}

//nolint:whitespace //editor/linter issue
func DeleteRaceBookingEntries(
	ctx context.Context,
	raceID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.BookingEntries.Delete(
		dm.Where(models.BookingEntries.Columns.RaceID.EQ(psql.Arg(raceID))),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteRaceResultEntries(
	ctx context.Context,
	raceID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	subQuery := psql.Select(
		sm.Columns(models.RaceGrids.Columns.ID),
		sm.From(models.RaceGrids.Name()),
		models.SelectWhere.RaceGrids.RaceID.EQ(raceID),
	)

	_, err := models.ResultEntries.Delete(
		dm.Where(models.ResultEntries.Columns.RaceGridID.In(subQuery)),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteRaceImportBatches(
	ctx context.Context,
	raceID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	subQuery := psql.Select(
		sm.Columns(models.RaceGrids.Columns.ID),
		sm.From(models.RaceGrids.Name()),
		models.SelectWhere.RaceGrids.RaceID.EQ(raceID),
	)

	_, err := models.ImportBatches.Delete(
		dm.Where(models.ImportBatches.Columns.RaceGridID.In(subQuery)),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteRaceRaceGrids(
	ctx context.Context,
	raceID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.RaceGrids.Delete(
		dm.Where(models.RaceGrids.Columns.RaceID.EQ(psql.Arg(raceID))),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteRaceGridRelated(
	ctx context.Context,
	raceGridID int32,
) error {
	if err := DeleteRaceGridBookingEntries(ctx, raceGridID); err != nil {
		return err
	}
	if err := DeleteRaceGridResultEntries(ctx, raceGridID); err != nil {
		return err
	}
	if err := DeleteRaceGridImportBatches(ctx, raceGridID); err != nil {
		return err
	}
	return nil
}

//nolint:whitespace //editor/linter issue
func DeleteRaceGridBookingEntries(
	ctx context.Context,
	raceGridID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.BookingEntries.Delete(
		dm.Where(models.BookingEntries.Columns.RaceGridID.EQ(psql.Arg(raceGridID))),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteRaceGridResultEntries(
	ctx context.Context,
	raceGridID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.ResultEntries.Delete(
		dm.Where(models.ResultEntries.Columns.RaceGridID.EQ(psql.Arg(raceGridID))),
	).Exec(ctx, executor)
	return err
}

//nolint:whitespace //editor/linter issue
func DeleteRaceGridImportBatches(
	ctx context.Context,
	raceGridID int32,
) error {
	var executor bob.Executor
	if executor = pgbob.FromContext(ctx); executor == nil {
		return ErrNoBobExecutorInContext
	}

	_, err := models.ImportBatches.Delete(
		dm.Where(models.ImportBatches.Columns.RaceGridID.EQ(psql.Arg(raceGridID))),
	).Exec(ctx, executor)
	return err
}
