package importsvc

import (
	"context"
	"encoding/json"

	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/types"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	mytypes "github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/conversion"
)

//nolint:whitespace,funlen,gocyclo // editor/linter issue
func (s *service) ComputeTeamBookingEntries(
	ctx context.Context,
	req *connect.Request[importv1.ComputeTeamBookingEntriesRequest],
) (
	*connect.Response[importv1.ComputeTeamBookingEntriesResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ComputeTeamBookingEntries")

	eventID := int32(req.Msg.GetEventId())

	// Load event to get season and current state.
	event, err := s.repo.Events().LoadByID(ctx, eventID)
	if err != nil {
		l.Error("failed to load event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	seasonID := event.SeasonID

	// Collect result entries across all grids for this event.
	grids, err := s.repo.Races().RaceGrids().LoadByEventID(ctx, eventID)
	if err != nil {
		l.Error("failed to load race grids", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load race grids")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	var resultEntries []*models.ResultEntry
	for _, grid := range grids {
		entries, loadErr := s.repo.ResultEntries().LoadByRaceGridID(ctx, grid.ID)
		if loadErr != nil {
			l.Error("failed to load result entries", log.ErrorField(loadErr))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load result entries")
			return nil, connect.NewError(s.conversion.MapErrorToRPCCode(loadErr), loadErr)
		}
		resultEntries = append(resultEntries, entries...)
	}

	// Build a driver-to-team lookup from the season's teams.
	teams, err := s.repo.Teams().Teams().LoadBySeasonID(ctx, seasonID)
	if err != nil {
		l.Error("failed to load teams", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load teams")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	driverToTeam := make(map[int32]int32)
	for _, team := range teams {
		teamDrivers, loadErr := s.repo.Teams().TeamDrivers().LoadByTeamID(ctx, team.ID)
		if loadErr != nil {
			l.Error("failed to load team drivers", log.ErrorField(loadErr))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load team drivers")
			return nil, connect.NewError(s.conversion.MapErrorToRPCCode(loadErr), loadErr)
		}
		for _, td := range teamDrivers {
			driverToTeam[td.DriverID] = team.ID
		}
	}

	fromState := event.ProcessingState
	toState := conversion.EventProcessingStateTeamEntriesComputed
	execUser := s.execUser(ctx)
	emptyJSON := types.JSON[json.RawMessage]{Val: json.RawMessage("{}")}

	var createdEntries int32

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		// Delete previously computed team booking entries for idempotency.
		if delErr := s.repo.BookingEntries().DeleteByEventIDAndTargetType(
			ctx, eventID, "team",
		); delErr != nil {
			return delErr
		}

		// Create one position-based team booking entry per result entry
		// whose driver is in a team.
		for _, entry := range resultEntries {
			if entry.DriverID.IsNull() {
				continue
			}
			driverID := entry.DriverID.GetOr(0)
			teamID, ok := driverToTeam[driverID]
			if !ok {
				continue
			}

			_, createErr := s.repo.BookingEntries().Create(
				ctx,
				&models.BookingEntrySetter{
					EventID:             omit.From(eventID),
					SourceResultEntryID: omitnull.From(entry.ID),
					TargetType:          omit.From(mytypes.TargetType("team")),
					TeamID:              omitnull.From(teamID),
					SourceType:          omit.From(mytypes.SourceType("position")),
					Points:              omit.From(int32(0)),
					Description:         omit.From("team position booking"),
					IsManual:            omit.From(false),
					MetadataJSON:        omit.From(emptyJSON),
					CreatedBy:           omit.From(execUser),
					UpdatedBy:           omit.From(execUser),
				})
			if createErr != nil {
				return createErr
			}
			createdEntries++
		}

		// Advance event processing state.
		_, updateErr := s.repo.Events().Update(ctx, eventID, &models.EventSetter{
			ProcessingState: omit.From(toState),
			UpdatedBy:       omit.From(execUser),
		})
		if updateErr != nil {
			return updateErr
		}

		// Write audit row.
		_, updateErr = s.repo.EventProcessingAudit().Create(
			ctx,
			&models.EventProcessingAuditSetter{
				EventID:     omit.From(eventID),
				FromState:   omitnull.From(fromState),
				ToState:     omit.From(toState),
				Action:      omit.From("compute_team_booking_entries"),
				PayloadJSON: omit.From(emptyJSON),
				CreatedBy:   omit.From(execUser),
				UpdatedBy:   omit.From(execUser),
			})
		return updateErr
	}); txErr != nil {
		l.Error("failed to compute team booking entries", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(
			codes.Error, "failed to compute team booking entries")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "team booking entries computed")
	return connect.NewResponse(&importv1.ComputeTeamBookingEntriesResponse{
		CreatedEntries: createdEntries,
	}), nil
}
