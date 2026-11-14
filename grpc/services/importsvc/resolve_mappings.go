package importsvc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect/v2"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/grpc/services/importsvc/importer"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
	"github.com/srlmgr/backend/log"
)

//nolint:whitespace,funlen,gocyclo // editor/linter issue
func (s *service) ResolveMappings(
	ctx context.Context,
	req *importv1.ResolveMappingsRequest,
) (*importv1.ResolveMappingsResponse, error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ResolveMappings")

	gridID := int32(req.GetRaceGridId())
	if gridID <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"race_grid_id is required")
	}

	// Load event to get current processing state.
	event, err := s.repo.Events().LoadByGridID(ctx, gridID)
	if err != nil {
		l.Error("failed to load event", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load event")
		return nil, s.conversion.ToConnectError(err)
	}
	ep := processor.NewEventProcInfoCollector(s.repo)
	epi, err := ep.ForEvent(ctx, event.ID)
	if err != nil {
		return nil, s.conversion.ToConnectError(err)
	}
	execUser := s.execUser(ctx)
	var resolved *importer.Result

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		batch, err := s.repo.ImportBatches().LoadByRaceGridID(ctx, gridID)
		if err != nil {
			return err
		}

		// Parsed rows provide simulation IDs and session data not stored in result entries.
		input, resolver, err := s.mappingInput(ctx, epi, batch, gridID)
		if err != nil {
			return err
		}

		existing, err := s.repo.ResultEntries().LoadByRaceGridID(ctx, gridID)
		if err != nil {
			return err
		}
		input, err = parsedInputWithResultEntries(input, existing, epi.Season.IsTeamBased)
		if err != nil {
			return err
		}
		resolved, err = resolver.ResolveInput(input)
		if err != nil {
			return fmt.Errorf("resolve mappings: %w", err)
		}
		if err := mergeResolvedEntries(resolved, existing); err != nil {
			return err
		}

		now := time.Now()
		for _, entry := range resolved.Entries {
			if entry == nil {
				continue
			}
			if entry.ID == 0 {
				return errors.New("result entry id is missing")
			}

			setter := buildResolvedResultEntrySetter(entry, now, execUser)
			if _, err := s.repo.ResultEntries().Update(ctx, entry.ID, setter); err != nil {
				return fmt.Errorf("update result entry %d: %w", entry.ID, err)
			}
		}

		for _, mapping := range resolved.Unmapped {
			if mapping.ResultIndex == nil {
				continue
			}
			index := int(mapping.GetResultIndex())
			if index < 0 || index >= len(existing) {
				return fmt.Errorf("unresolved mapping has invalid result index %d", index)
			}
			entryID := uint32(existing[index].ID)
			mapping.ResultEntryId = &entryID
		}
		scopeMappingsToGrid(resolved.Unmapped, uint32(gridID))
		return nil
	}); txErr != nil {
		l.Error("failed to resolve mappings", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to resolve mappings")
		return nil, s.conversion.ToConnectError(txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "mappings resolved")
	return &importv1.ResolveMappingsResponse{
		Rows:               s.resultEntryRows(resolved.Entries),
		UnresolvedMappings: resolved.Unmapped,
	}, nil
}

//nolint:whitespace // editor/linter issue
func parsedInputWithResultEntries(
	input *importer.ParsedImportPayload,
	entries []*models.ResultEntry,
	isTeamBased bool,
) (*importer.ParsedImportPayload, error) {
	if input == nil {
		return nil, errors.New("parsed import input is nil")
	}
	if len(input.Results) != len(entries) {
		return nil, fmt.Errorf(
			"result entry count %d does not match import row count %d",
			len(entries),
			len(input.Results),
		)
	}

	entriesInImportOrder := append([]*models.ResultEntry(nil), entries...)
	for i, entry := range entriesInImportOrder {
		if entry == nil {
			return nil, fmt.Errorf("result entry %d is nil", i)
		}
	}
	sort.SliceStable(entriesInImportOrder, func(i, j int) bool {
		return entriesInImportOrder[i].ID < entriesInImportOrder[j].ID
	})
	sourceRows := make(map[*models.ResultEntry]*importer.ResultRow, len(entries))
	for i, entry := range entriesInImportOrder {
		if input.Results[i] == nil {
			return nil, fmt.Errorf("import result row %d is nil", i)
		}
		sourceRows[entry] = input.Results[i]
	}

	updated := *input
	updated.Results = make([]*importer.ResultRow, len(entries))
	for i, entry := range entries {
		updated.Results[i] = resultEntryImportRow(sourceRows[entry], entry, isTeamBased)
	}
	return &updated, nil
}

//nolint:whitespace // editor/linter issue
func resultEntryImportRow(
	sourceRow *importer.ResultRow,
	entry *models.ResultEntry,
	isTeamBased bool,
) *importer.ResultRow {
	row := *sourceRow
	row.FinPos = int(entry.FinishPosition)
	row.Car = entry.RawCarName.GetOr("")
	row.CarNumber = entry.CarNumber.GetOr("")
	row.StartPos = int(entry.StartPosition.GetOrZero())
	row.QualiLapTime = int(entry.QualiLapTimeMS.GetOrZero())
	row.TotalTime = int(entry.TotalTimeMS.GetOrZero())
	row.FastestLapTime = int(entry.FastestLapTimeMS.GetOrZero())
	row.Laps = int(entry.LapsCompleted)
	row.Incidents = int(entry.Incidents.GetOrZero())
	row.Offtracks = int(entry.Offtracks)
	if isTeamBased {
		name := entry.RawTeamName.GetOr("")
		if row.Name != name {
			row.TeamID = ""
		}
		row.Name = name
	} else {
		name := entry.RawDriverName.GetOr("")
		if row.Name != name {
			row.DriverID = ""
		}
		row.Name = name
	}
	if row.Car != sourceRow.Car {
		row.CarID = ""
	}
	return &row
}

//nolint:whitespace // editor/linter issue
func mergeResolvedEntries(
	result *importer.Result,
	existing []*models.ResultEntry,
) error {
	if result == nil {
		return errors.New("resolved result is nil")
	}
	if len(result.Entries) != len(existing) {
		return fmt.Errorf(
			"resolved entry count %d does not match existing entry count %d",
			len(result.Entries),
			len(existing),
		)
	}

	resolvedEntries := result.Entries
	mergedEntries := make([]*models.ResultEntry, 0, len(existing))
	for i := 0; len(existing) > 0; i++ {
		existingEntry := existing[0]
		if existingEntry == nil {
			return fmt.Errorf("existing result entry %d is nil", i)
		}
		if len(resolvedEntries) == 0 {
			return fmt.Errorf("resolved result entry %d is missing", i)
		}
		resolved := resolvedEntries[0]
		if resolved == nil {
			return fmt.Errorf("resolved result entry %d is nil", i)
		}

		entry := *existingEntry
		entry.State = resolved.State
		entry.DriverID = resolved.DriverID
		entry.TeamID = resolved.TeamID
		entry.TeamDrivers = resolved.TeamDrivers
		entry.CarModelVariantID = resolved.CarModelVariantID
		entry.CarClassID = resolved.CarClassID
		mergedEntries = append(mergedEntries, &entry)
		existing = existing[1:]
		resolvedEntries = resolvedEntries[1:]
	}
	if len(resolvedEntries) > 0 {
		return errors.New("resolved result has extra entries")
	}
	result.Entries = mergedEntries
	return nil
}

//nolint:whitespace // editor/linter issue
func buildResolvedResultEntrySetter(
	entry *models.ResultEntry,
	now time.Time,
	execUser string,
) *models.ResultEntrySetter {
	return &models.ResultEntrySetter{
		State:             omit.From(entry.State),
		DriverID:          omitnull.FromNull(entry.DriverID),
		TeamID:            omitnull.FromNull(entry.TeamID),
		TeamDrivers:       omitnull.FromNull(entry.TeamDrivers),
		CarModelVariantID: omitnull.FromNull(entry.CarModelVariantID),
		CarClassID:        omitnull.FromNull(entry.CarClassID),
		UpdatedAt:         omit.From(now),
		UpdatedBy:         omit.From(execUser),
	}
}
