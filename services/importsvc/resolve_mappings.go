package importsvc

import (
	"context"
	"errors"
	"fmt"
	"time"

	importv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/import/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

//nolint:whitespace,funlen,gocyclo // editor/linter issue
func (s *service) ResolveMappings(
	ctx context.Context,
	req *connect.Request[importv1.ResolveMappingsRequest],
) (
	*connect.Response[importv1.ResolveMappingsResponse], error,
) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ResolveMappings")

	importBatchID := int32(req.Msg.GetImportBatchId())
	if importBatchID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("import_batch_id is required"))
	}

	execUser := s.execUser(ctx)
	var resolvedMappings int32

	if txErr := s.withTx(ctx, func(ctx context.Context) error {
		batch, err := s.repo.ImportBatches().LoadByID(ctx, importBatchID)
		if err != nil {
			return err
		}

		event, err := s.repo.Events().LoadByID(ctx, batch.EventID)
		if err != nil {
			return err
		}

		importProcessor, simulation, err := s.resolveProcessorForEvent(ctx, event)
		if err != nil {
			return err
		}

		importFormat := string(batch.ImportFormat)
		if !processor.SupportsFormat(importProcessor, importFormat) {
			return fmt.Errorf(
				"%w: simulation=%q format=%q",
				processor.ErrUnsupportedFormat,
				simulation.Name,
				importFormat,
			)
		}

		input, err := importProcessor.Process(ctx, importFormat, batch.Payload)
		if err != nil {
			return fmt.Errorf("process import payload: %w", err)
		}

		existing, err := s.repo.ResultEntries().LoadByImportBatchID(ctx, importBatchID)
		if err != nil {
			return err
		}

		mappingErrorsBefore := countMappingErrors(existing)

		resolver := processor.NewResolver(
			processor.NewRepositoryEntityResolver(s.repo, simulation),
		)
		resolved, err := resolver.ResolveNonMapped(input, existing)
		if err != nil {
			return fmt.Errorf("resolve mappings: %w", err)
		}

		now := time.Now()
		for _, entry := range resolved.Entries {
			if entry == nil {
				continue
			}
			if entry.ID == 0 {
				return errors.New("result entry id is missing")
			}

			setter := &models.ResultEntrySetter{
				State:     omit.From(entry.State),
				UpdatedAt: omit.From(now),
				UpdatedBy: omit.From(execUser),
			}
			if !entry.DriverID.IsNull() {
				setter.DriverID = omitnull.From(entry.DriverID.GetOr(0))
			}
			if !entry.CarModelID.IsNull() {
				setter.CarModelID = omitnull.From(entry.CarModelID.GetOr(0))
			}

			if _, err := s.repo.ResultEntries().Update(ctx, entry.ID, setter); err != nil {
				return fmt.Errorf("update result entry %d: %w", entry.ID, err)
			}
		}

		mappingErrorsAfter := countMappingErrors(resolved.Entries)
		if mappingErrorsBefore > mappingErrorsAfter {
			resolvedMappings = int32(mappingErrorsBefore - mappingErrorsAfter)
		}

		return nil
	}); txErr != nil {
		l.Error("failed to resolve mappings", log.ErrorField(txErr))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to resolve mappings")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(txErr), txErr)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "mappings resolved")
	return connect.NewResponse(&importv1.ResolveMappingsResponse{
		ResolvedMappings: resolvedMappings,
	}), nil
}

func countMappingErrors(entries []*models.ResultEntry) int {
	count := 0
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if entry.State == conversion.ResultStateMappingError {
			count++
		}
	}
	return count
}
