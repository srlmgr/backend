package importsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/samber/lo"
	"github.com/stephenafamo/bob/types"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	mytypes "github.com/srlmgr/backend/db/mytypes"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/services/importsvc/points"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

//nolint:whitespace,funlen // editor/linter issue
func (s *service) ComputeBookingEntries(
	ctx context.Context,
	req *connect.Request[importv1.ComputeBookingEntriesRequest],
) (
	*connect.Response[importv1.ComputeBookingEntriesResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ComputeBookingEntries (auto detect)")

	eventID := int32(req.Msg.GetEventId())

	// Load the event to capture current processing state.
	event, err := s.repo.Events().LoadByID(ctx, eventID)
	if err != nil {
		l.Error("failed to load event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	// Collect result entries across all races for this event.
	raceGrids, err := s.repo.Races().RaceGrids().LoadByEventID(ctx, eventID)
	if err != nil {
		l.Error("failed to load race grids", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load race grids")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	var resultEntries []*models.ResultEntry
	for _, grid := range raceGrids {
		entries, loadErr := s.repo.ResultEntries().LoadByRaceGridID(ctx, grid.ID)
		if loadErr != nil {
			l.Error("failed to load result entries", log.ErrorField(loadErr))
			trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load result entries")
			return nil, connect.NewError(s.conversion.MapErrorToRPCCode(loadErr), loadErr)
		}
		resultEntries = append(resultEntries, entries...)
	}
	raceIDByGridID := lo.SliceToMap(raceGrids, func(item *models.RaceGrid) (int32, int32) {
		return item.ID, item.RaceID
	})

	fromState := event.ProcessingState
	toState := conversion.EventProcessingStateDriverEntriesComputed
	execUser := s.execUser(ctx)
	emptyJSON := types.JSON[json.RawMessage]{Val: json.RawMessage("{}")}

	var epi *processor.EventProcInfo
	ep := processor.NewEventProcInfoCollector(s.repo)
	epi, err = ep.ForEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	var createdEntries int32

	outputs, err := s.autoComputeEvent(ctx, resultEntries, epi)
	if err != nil {
		l.Error("failed to compute event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to compute event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	l.Debug("event computed", log.Int("num_outputs", len(outputs)))
	lo.ForEach(outputs, func(output points.GridOutput, _ int) {
		fmt.Printf("summary for gridID %d\n", output.GridID)
		lo.ForEach(output.Outputs, func(entry points.Output, _ int) {
			fmt.Printf(
				"entry for refID %d: points=%.1f, origin:%s description=%s\n",
				entry.ReferenceID(), entry.Points(), entry.Origin(), entry.Msg(),
			)
		})
	})

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		// Delete previously computed driver booking entries for idempotency.
		if delErr := s.repo.BookingEntries().DeleteNonManualByEvent(
			ctx, eventID,
		); delErr != nil {
			return delErr
		}

		// create primary booking entries. these are fine granular entries
		for i := range outputs {
			gridOutput := outputs[i]
			for j := range gridOutput.Outputs {
				output := gridOutput.Outputs[j]
				setter := &models.BookingEntrySetter{
					EventID:      omit.From(eventID),
					RaceID:       omitnull.From(raceIDByGridID[gridOutput.GridID]),
					RaceGridID:   omitnull.From(gridOutput.GridID),
					SourceType:   omit.From(mytypes.SourceType(output.Origin().String())),
					Points:       omit.From(int32(output.Points())),
					Description:  omit.From(output.Msg()),
					IsManual:     omit.From(false),
					MetadataJSON: omit.From(emptyJSON),
					CreatedBy:    omit.From(execUser),
					UpdatedBy:    omit.From(execUser),
				}
				if epi.Season.IsTeamBased {
					setter.TeamID = omitnull.From(output.ReferenceID())
					setter.TargetType = omit.From(mytypes.TargetType("team"))
				} else {
					setter.DriverID = omitnull.From(output.ReferenceID())
					setter.TargetType = omit.From(mytypes.TargetType("driver"))
				}
				_, createErr := s.repo.BookingEntries().Create(ctx, setter)
				if createErr != nil {
					return createErr
				}
				createdEntries++
			}
		}
		if err := s.createSecondaryEntries(ctx, epi, outputs, execUser); err != nil {
			return err
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
				Action:      omit.From("compute_driver_booking_entries"),
				PayloadJSON: omit.From(emptyJSON),
				CreatedBy:   omit.From(execUser),
				UpdatedBy:   omit.From(execUser),
			})
		return updateErr
	}); txErr != nil {
		l.Error("failed to compute booking entries", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).
			SetStatus(codes.Error, "failed to compute booking entries")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "booking entries computed")
	return connect.NewResponse(&importv1.ComputeBookingEntriesResponse{
		CreatedEntries: createdEntries,
	}), nil
}

func (s *service) createSecondaryEntries(
	ctx context.Context,
	epi *processor.EventProcInfo,
	outputs []points.GridOutput,
	execUser string,
) error {
	// create "secondary" entries by aggregating the "primary" entries
	if epi.Season.IsTeamBased {
		return s.createSecondaryDriverEntries(ctx, epi, outputs, execUser)
	} else if epi.Season.HasTeams {
		return s.createSecondaryTeamEntries(ctx, epi, outputs, execUser)
	}

	return nil
}

//nolint:whitespace // editor/linter issue,
func (s *service) createSecondaryDriverEntries(
	ctx context.Context,
	epi *processor.EventProcInfo,
	outputs []points.GridOutput,
	execUser string,
) error {
	return nil
}

// used as secondary when season is driver based
//
//nolint:funlen,whitespace // editor/linter issue,
func (s *service) createSecondaryTeamEntries(
	ctx context.Context,
	epi *processor.EventProcInfo,
	outputs []points.GridOutput,
	execUser string,
) error {
	byDriver := lo.GroupBy(s.combineOutputs(outputs), func(output points.Output) int32 {
		return output.ReferenceID()
	})

	type _driverData struct {
		driverID int32
		points   float64
	}
	driverData := make([]_driverData, 0)
	for driverID := range byDriver {
		entries := byDriver[driverID]
		pointsSum := lo.SumBy(entries, func(entry points.Output) float64 {
			return float64(entry.Points())
		})
		driverData = append(driverData, _driverData{
			driverID: driverID,
			points:   pointsSum,
		})
	}
	// sort drivers by points desc
	slices.SortFunc(driverData, func(a, b _driverData) int {
		return int(b.points - a.points)
	})
	teamDrivers, err := s.repo.Queries().QueryTeamDrivers().FindBySeason(
		ctx, epi.Season.ID)
	if err != nil {
		return err
	}
	teamIDByDriverID := lo.SliceToMap(teamDrivers,
		func(item *models.TeamDriver) (int32, int32) {
			return item.DriverID, item.TeamID
		})
	settersByTeamID := make(map[int32][]*models.BookingEntrySetter)
	for i := range driverData {
		data := driverData[i]
		teamID := teamIDByDriverID[data.driverID]
		if len(settersByTeamID[teamID]) < int(epi.Season.TeamPointsTopN.GetOrZero()) {
			settersByTeamID[teamID] = append(settersByTeamID[teamID],
				&models.BookingEntrySetter{
					EventID:    omit.From(epi.Event.ID),
					TeamID:     omitnull.From(teamID),
					SourceType: omit.From(mytypes.SourceType("team_contribution")),
					TargetType: omit.From(mytypes.TargetType("team")),
					Points:     omit.From(int32(data.points)),
					Description: omit.From(
						fmt.Sprintf("contribution of driverID %d", data.driverID),
					),
					IsManual:  omit.From(false),
					CreatedBy: omit.From(execUser),
					UpdatedBy: omit.From(execUser),
				})
		}
	}
	for k, v := range settersByTeamID {
		fmt.Printf("teamID %d has %d contributing drivers\n", k, len(v))
		for _, setter := range v {
			_, createErr := s.repo.BookingEntries().Create(ctx, setter)
			if createErr != nil {
				return createErr
			}
		}
	}

	return nil
}

func (s *service) combineOutputs(outputs []points.GridOutput) []points.Output {
	ret := make([]points.Output, 0)
	for i := range outputs {
		gridOutput := outputs[i]
		ret = append(ret, gridOutput.Outputs...)
	}
	return ret
}

//nolint:whitespace // editor/linter issue,
func (s *service) primaryReferenceID(
	entry *models.ResultEntry,
	epi *processor.EventProcInfo,
) int32 {
	if epi.Season.IsTeamBased {
		return entry.TeamID.GetOrZero()
	}
	return entry.DriverID.GetOrZero()
}

//nolint:whitespace // editor/linter issue
func (s *service) autoComputeEvent(
	ctx context.Context,
	entries []*models.ResultEntry,
	epi *processor.EventProcInfo,
) ([]points.GridOutput, error) {
	pe := points.NewEventProcessor(epi.PointSystemSettings)
	conv := points.NewConverter()
	byGridID := lo.GroupBy(entries, func(item *models.ResultEntry) int32 {
		return item.RaceGridID
	})
	gridInputs := make([]points.GridInput, 0)

	for gridID := range byGridID {
		entries := byGridID[gridID]
		inps := lo.Map(entries, func(item *models.ResultEntry, _ int) points.Input {
			ret := conv.ResultEntryToInput(item,
				points.WithReferenceID(s.primaryReferenceID(item, epi)),
			)
			return ret
		})

		gridInputs = append(gridInputs, points.GridInput{
			GridID: gridID,
			Inputs: inps,
		})
	}

	return pe.ProcessAll(ctx, gridInputs, epi.ResolverFunc(ctx))
}
