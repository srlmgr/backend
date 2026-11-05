package importsvc

import (
	"context"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
)

//nolint:whitespace // editor/linter issue
func (s *service) GetPreprocessPreview(
	ctx context.Context,
	req *importv1.GetPreprocessPreviewRequest,
) (*importv1.GetPreprocessPreviewResponse, error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetPreprocessPreview")

	gridID := int32(req.GetRaceGridId())

	// Load result entries for the race.
	entries, err := s.repo.ResultEntries().LoadByRaceGridID(ctx, gridID)
	if err != nil {
		l.Error("failed to load result entries", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load result entries")
		return nil, s.conversion.ToConnectError(err)
	}

	// Convert result entries to proto.
	rows := make([]*commonv1.ResultEntry, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, s.conversion.ResultEntryToResultEntry(entry))
	}

	// Build unresolved mappings from entries missing canonical IDs.
	var unresolvedMappings []*commonv1.UnresolvedMapping
	for _, entry := range entries {
		if entry.DriverID.IsNull() && entry.RawDriverName.GetOr("") != "" {
			unresolvedMappings = append(unresolvedMappings, &commonv1.UnresolvedMapping{
				SourceValue: entry.RawDriverName.GetOr(""),
				MappingType: "driver",
			})
		}
		if entry.CarModelVariantID.IsNull() && entry.RawCarName.GetOr("") != "" {
			unresolvedMappings = append(unresolvedMappings, &commonv1.UnresolvedMapping{
				SourceValue: entry.RawCarName.GetOr(""),
				MappingType: "car_model_variant",
			})
		}
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "preprocess preview loaded")
	return &importv1.GetPreprocessPreviewResponse{
		Rows:               rows,
		UnresolvedMappings: unresolvedMappings,
	}, nil
}
