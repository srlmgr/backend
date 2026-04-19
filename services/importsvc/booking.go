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

//nolint:whitespace,funlen,gocyclo // editor/linter issue, lots of work to do here
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

	fromState := event.ProcessingState
	toState := conversion.EventProcessingStateDriverEntriesComputed
	execUser := s.execUser(ctx)

	var epi *processor.EventProcInfo
	ep := processor.NewEventProcInfoCollector(s.repo)
	epi, err = ep.ForEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	teamDrivers, err := s.repo.Queries().QueryTeamDrivers().FindBySeason(
		ctx, epi.Season.ID)
	if err != nil {
		teamDrivers = nil // not critical, we can proceed without team driver info
	}
	penalties := s.loadPenalties(ctx, eventID)
	bookingProc := newBookingProc(
		epi,
		resultEntries,
		raceGrids,
		teamDrivers,
		penalties,
		execUser)

	var createdEntries int32

	outputs, err := bookingProc.computeEvent(ctx)
	if err != nil {
		l.Error("failed to compute event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to compute event")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}
	// TODO: remove debugs
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

		pBookings, err := bookingProc.computeBookingEntries(ctx)
		if err != nil {
			return err
		}

		dbgSingle := false
		if dbgSingle {
			for i, setter := range pBookings {
				_, err = s.repo.BookingEntries().Create(ctx, setter)
				if err != nil {
					log.Error("failed to create booking entry",
						log.ErrorField(err), log.Int("index", i))
					return err
				}
			}
		} else {
			_, err = s.repo.BookingEntries().CreateMany(ctx, pBookings)
			if err != nil {
				return err
			}
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
				EventID:   omit.From(eventID),
				FromState: omitnull.From(fromState),
				ToState:   omit.From(toState),
				Action:    omit.From("compute_driver_booking_entries"),

				CreatedBy: omit.From(execUser),
				UpdatedBy: omit.From(execUser),
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

//nolint:whitespace // editor/linter issue
func (s *service) loadPenalties(
	ctx context.Context,
	eventID int32,
) []*models.BookingEntry {
	entries, err := s.repo.BookingEntries().LoadByEventIDAndSourceType(
		ctx, eventID, points.PointsPolicyPenalty.String(),
	)
	if err != nil {
		return nil
	}
	return entries
}

type (
	bookingProc struct {
		resultEntries  []*models.ResultEntry
		raceGrids      []*models.RaceGrid
		raceIDByGridID map[int32]int32
		teamDrivers    []*models.TeamDriver
		penalties      []*models.BookingEntry
		epi            *processor.EventProcInfo
		execUser       string
	}
)

//nolint:whitespace // editor/linter issue
func newBookingProc(
	epi *processor.EventProcInfo,
	resultEntries []*models.ResultEntry,
	raceGrids []*models.RaceGrid,
	teamDrivers []*models.TeamDriver,
	penalties []*models.BookingEntry,
	execUser string,
) *bookingProc {
	return &bookingProc{
		resultEntries: resultEntries,
		raceGrids:     raceGrids,
		teamDrivers:   teamDrivers,
		penalties:     penalties,
		raceIDByGridID: lo.SliceToMap(raceGrids,
			func(item *models.RaceGrid) (int32, int32) {
				return item.ID, item.RaceID
			}),
		epi:      epi,
		execUser: execUser,
	}
}

//nolint:whitespace // editor/linter issue
func (s *bookingProc) computeEvent(
	ctx context.Context,
) ([]points.GridOutput, error) {
	pe := points.NewEventProcessor(s.epi.PointSystemSettings)
	conv := points.NewConverter()
	byGridID := lo.GroupBy(s.resultEntries, func(item *models.ResultEntry) int32 {
		return item.RaceGridID
	})
	gridInputs := make([]points.GridInput, 0)

	for gridID := range byGridID {
		entries := byGridID[gridID]
		inps := lo.Map(entries, func(item *models.ResultEntry, _ int) points.Input {
			ret := conv.ResultEntryToInput(item,
				points.WithReferenceID(s.primaryReferenceID(item, s.epi)),
			)
			return ret
		})

		gridInputs = append(gridInputs, points.GridInput{
			GridID: gridID,
			Inputs: inps,
		})
	}

	return pe.ProcessAll(ctx, gridInputs, s.epi.ResolverFunc(ctx))
}

//nolint:whitespace // editor/linter issue
func (s *bookingProc) computeBookingEntries(
	ctx context.Context,
) ([]*models.BookingEntrySetter, error) {
	rawOutputs, err := s.computeEvent(ctx)
	if err != nil {
		return nil, err
	}
	setters := make([]*models.BookingEntrySetter, 0)
	pEntries := s.computePrimaryBookings(rawOutputs)
	setters = append(setters, pEntries...)

	// add penalties (needs to be done before computing secondary entries)

	if sEntries, err := s.computeSecondaryBookings(rawOutputs); err != nil {
		return nil, err
	} else {
		setters = append(setters, sEntries...)
	}

	return setters, nil
}

func (s *bookingProc) computePrimaryBookings(
	outputs []points.GridOutput,
) []*models.BookingEntrySetter {
	// create primary booking entries. these are fine granular entries
	setters := make([]*models.BookingEntrySetter, 0)
	for i := range outputs {
		gridOutput := outputs[i]
		for j := range gridOutput.Outputs {
			output := gridOutput.Outputs[j]
			setter := s.baseSetter(output)
			setter.RaceID = omitnull.From(s.raceIDByGridID[gridOutput.GridID])
			setter.RaceGridID = omitnull.From(gridOutput.GridID)

			if s.epi.Season.IsTeamBased {
				setter.TeamID = omitnull.From(output.ReferenceID())
				setter.TargetType = omit.From(mytypes.TargetType("team"))
			} else {
				setter.DriverID = omitnull.From(output.ReferenceID())
				setter.TargetType = omit.From(mytypes.TargetType("driver"))
			}
			setters = append(setters, setter)
		}
	}
	return setters
}

func (s *bookingProc) computeSecondaryBookings(
	outputs []points.GridOutput,
) ([]*models.BookingEntrySetter, error) {
	// create "secondary" entries by aggregating the "primary" entries
	if s.epi.Season.IsTeamBased {
		return s.createSecondaryDriverEntries(outputs)
	} else if s.epi.Season.HasTeams {
		return s.createSecondaryTeamEntries(outputs)
	}

	return nil, nil
}

// used as secondary when season is driver based
// we calc the sum of points for each driver and use the top N drivers of each team
// to create booking entries for the team standings
//
//nolint:funlen,whitespace // editor/linter issue,
func (s *bookingProc) createSecondaryTeamEntries(
	outputs []points.GridOutput,
) ([]*models.BookingEntrySetter, error) {
	byDriver := lo.GroupBy(s.combineOutputs(outputs), func(output points.Output) int32 {
		return output.ReferenceID()
	})
	penByDriver := lo.GroupBy(s.penalties, func(be *models.BookingEntry) int32 {
		return be.DriverID.GetOrZero()
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
		// process penalties (note: penalty points are stored as negative values))
		if penEntries, ok := penByDriver[driverID]; ok {
			pointsSum += lo.SumBy(penEntries, func(entry *models.BookingEntry) float64 {
				return float64(entry.Points)
			})
		}
		driverData = append(driverData, _driverData{
			driverID: driverID,
			points:   pointsSum,
		})
	}
	// sort drivers by points desc
	slices.SortFunc(driverData, func(a, b _driverData) int {
		return int(b.points - a.points)
	})

	teamIDByDriverID := lo.SliceToMap(s.teamDrivers,
		func(item *models.TeamDriver) (int32, int32) {
			return item.DriverID, item.TeamID
		})
	settersByTeamID := make(map[int32][]*models.BookingEntrySetter)
	for i := range driverData {
		data := driverData[i]
		teamID, ok := teamIDByDriverID[data.driverID]
		if !ok {
			continue // driver not assigned to team, skip
		}
		if len(settersByTeamID[teamID]) < int(s.epi.Season.TeamPointsTopN.GetOrZero()) {
			settersByTeamID[teamID] = append(settersByTeamID[teamID],
				&models.BookingEntrySetter{
					EventID:    omit.From(s.epi.Event.ID),
					TeamID:     omitnull.From(teamID),
					SourceType: omit.From(mytypes.SourceType("team_contribution")),
					TargetType: omit.From(mytypes.TargetType("team")),
					Points:     omit.From(int32(data.points)),
					Description: omit.From(
						fmt.Sprintf("contribution of driverID %d", data.driverID),
					),
					IsManual:  omit.From(false),
					CreatedBy: omit.From(s.execUser),
					UpdatedBy: omit.From(s.execUser),
				})
		}
	}

	return lo.FlatMap(lo.Values(settersByTeamID),
		func(setters []*models.BookingEntrySetter, _ int) []*models.BookingEntrySetter {
			return setters
		}), nil
}

// used as secondary when season is team based
// here we just duplicate the booking entries of a team for each team driver
// in that event
//
//nolint:whitespace // editor/linter issue,
func (s *bookingProc) createSecondaryDriverEntries(
	outputs []points.GridOutput,
) ([]*models.BookingEntrySetter, error) {
	setters := make([]*models.BookingEntrySetter, 0)
	for i := range outputs {
		gridOutput := outputs[i]
		gridRE := lo.Filter(s.resultEntries, func(item *models.ResultEntry, _ int) bool {
			return item.RaceGridID == gridOutput.GridID
		})
		byTeamID := lo.SliceToMap(gridRE,
			func(item *models.ResultEntry) (int32, *models.ResultEntry) {
				return item.TeamID.GetOrZero(), item
			})
		for j := range gridOutput.Outputs {
			item := gridOutput.Outputs[j]
			if resEntry, ok := byTeamID[item.ReferenceID()]; ok {
				for _, td := range resEntry.TeamDrivers.GetOrZero().DriverIDs {
					newEntry := s.baseSetter(item)
					newEntry.RaceID = omitnull.From(s.raceIDByGridID[gridOutput.GridID])
					newEntry.RaceGridID = omitnull.From(gridOutput.GridID)
					newEntry.DriverID = omitnull.From(td)
					newEntry.TargetType = omit.From(mytypes.TargetType("driver"))

					setters = append(setters, newEntry)
				}
			}

		}

	}
	return setters, nil
}

func (s *bookingProc) combineOutputs(outputs []points.GridOutput) []points.Output {
	ret := make([]points.Output, 0)
	for i := range outputs {
		gridOutput := outputs[i]
		ret = append(ret, gridOutput.Outputs...)
	}
	return ret
}

//nolint:whitespace // editor/linter issue,
func (s *bookingProc) primaryReferenceID(
	entry *models.ResultEntry,
	epi *processor.EventProcInfo,
) int32 {
	if epi.Season.IsTeamBased {
		return entry.TeamID.GetOrZero()
	}
	return entry.DriverID.GetOrZero()
}

func (s *bookingProc) baseSetter(output points.Output) *models.BookingEntrySetter {
	emptyJSON := types.JSON[json.RawMessage]{Val: json.RawMessage("{}")}
	createJSON := func(data any) types.JSON[json.RawMessage] {
		if data == nil {
			return emptyJSON
		}
		b, err := json.Marshal(data)
		if err != nil {
			return emptyJSON
		}
		return types.JSON[json.RawMessage]{Val: json.RawMessage(b)}
	}
	setter := &models.BookingEntrySetter{
		EventID:      omit.From(s.epi.Event.ID),
		SourceType:   omit.From(mytypes.SourceType(output.Origin().String())),
		Points:       omit.From(int32(output.Points())),
		Description:  omit.From(output.Msg()),
		IsManual:     omit.From(false),
		MetadataJSON: omit.From(createJSON(output.Meta())),
		CreatedBy:    omit.From(s.execUser),
		UpdatedBy:    omit.From(s.execUser),
	}
	return setter
}
