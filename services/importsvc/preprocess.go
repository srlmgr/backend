package importsvc

import (
	"context"
	"encoding/json"
	"time"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/types"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

//nolint:whitespace // editor/linter issue
func (s *service) GetPreprocessPreview(
	ctx context.Context,
	req *connect.Request[importv1.GetPreprocessPreviewRequest],
) (
	*connect.Response[importv1.GetPreprocessPreviewResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetPreprocessPreview")

	eventID := int32(req.Msg.GetEventId())
	raceID := int32(req.Msg.GetRaceId())

	// Resolve the latest import batch.
	batch, err := s.repo.ImportBatches().LoadLatestByEventIDAndRaceID(ctx, eventID, raceID)
	if err != nil {
		l.Error("failed to load import batch", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load import batch")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	// Load result entries for the batch.
	entries, err := s.repo.ResultEntries().LoadByImportBatchID(ctx, batch.ID)
	if err != nil {
		l.Error("failed to load result entries", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load result entries")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	fromState := batch.ProcessingState
	toState := "preprocessed"
	execUser := s.execUser(ctx)
	emptyJSON := types.JSON[json.RawMessage]{V: json.RawMessage("{}")}

	// Transition state to preprocessed.
	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		_, updateErr := s.repo.ImportBatches().Update(ctx, batch.ID, &models.ImportBatchSetter{
			ProcessingState: omit.From(toState),
			ProcessedAt:     omitnull.From(time.Now()),
			UpdatedAt:       omit.From(time.Now()),
			UpdatedBy:       omit.From(execUser),
		})
		if updateErr != nil {
			return updateErr
		}

		_, updateErr = s.repo.Events().Update(ctx, eventID, &models.EventSetter{
			ProcessingState: omit.From(toState),
			UpdatedAt:       omit.From(time.Now()),
			UpdatedBy:       omit.From(execUser),
		})
		if updateErr != nil {
			return updateErr
		}

		_, updateErr = s.repo.EventProcessingAudit().Create(ctx, &models.EventProcessingAuditSetter{
			EventID:       omit.From(eventID),
			ImportBatchID: omitnull.From(batch.ID),
			FromState:     omitnull.From(fromState),
			ToState:       omit.From(toState),
			Action:        omit.From("get_preprocess_preview"),
			PayloadJSON:   omit.From(emptyJSON),
			CreatedBy:     omit.From(execUser),
			UpdatedBy:     omit.From(execUser),
		})
		return updateErr
	}); txErr != nil {
		l.Error("failed to transition to preprocessed state", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to transition to preprocessed state")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	// Convert result entries to proto.
	rows := make([]*commonv1.ResultEntry, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, s.conversion.ResultEntryToResultEntry(entry))
	}

	// Build unresolved mappings from entries missing canonical IDs.
	var unresolvedMappings []*commonv1.UnresolvedMapping
	for _, entry := range entries {
		if !entry.DriverID.IsValid() && entry.DriverName != "" {
			unresolvedMappings = append(unresolvedMappings, &commonv1.UnresolvedMapping{
				SourceValue: entry.DriverName,
				MappingType: "driver",
			})
		}
		if !entry.CarModelID.IsValid() && entry.CarName.GetOr("") != "" {
			unresolvedMappings = append(unresolvedMappings, &commonv1.UnresolvedMapping{
				SourceValue: entry.CarName.GetOr(""),
				MappingType: "car_model",
			})
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "preprocess preview loaded")
	return connect.NewResponse(&importv1.GetPreprocessPreviewResponse{
		Rows:               rows,
		UnresolvedMappings: unresolvedMappings,
	}), nil
}
