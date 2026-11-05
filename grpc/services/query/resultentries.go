package query

import (
	"context"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
)

// GetResultEntry returns a result entry by ID.
//
//nolint:whitespace,dupl // editor/linter issue
func (s *service) GetResultEntry(
	ctx context.Context,
	req *queryv1.GetResultEntryRequest,
) (*queryv1.GetResultEntryResponse, error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("GetResultEntry", log.Uint32("id", req.GetResultEntryId()))

	item, err := s.repo.ResultEntries().LoadByID(ctx, int32(req.GetResultEntryId()))
	if err != nil {
		l.Error("failed to load result entry", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load result entry")
		return nil, s.conversion.ToConnectError(err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "result entry loaded")
	return &queryv1.GetResultEntryResponse{
		ResultEntry: s.conversion.ResultEntryToResultEntry(item),
	}, nil
}
