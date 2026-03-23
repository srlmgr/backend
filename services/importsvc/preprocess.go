package importsvc

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
)

//nolint:whitespace,funlen // editor/linter issue
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

	// Convert result entries to proto.
	rows := make([]*commonv1.ResultEntry, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, s.conversion.ResultEntryToResultEntry(entry))
	}

	// Build unresolved mappings from entries missing canonical IDs.
	var unresolvedMappings []*commonv1.UnresolvedMapping
	for _, entry := range entries {
		if entry.DriverID.IsNull() && entry.DriverName != "" {
			unresolvedMappings = append(unresolvedMappings, &commonv1.UnresolvedMapping{
				SourceValue: entry.DriverName,
				MappingType: "driver",
			})
		}
		if entry.CarModelID.IsNull() && entry.CarName.GetOr("") != "" {
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
