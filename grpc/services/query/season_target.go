package query

import (
	"context"
	"fmt"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
)

const maxSeasonTargetValue = uint32(1<<31 - 1)

// ResolveSeasonTarget resolves human-friendly sequence numbers to database IDs.
//
//nolint:whitespace // editor/linter issue
func (s *service) ResolveSeasonTarget(
	ctx context.Context,
	req *connect.Request[queryv1.ResolveSeasonTargetRequest],
) (*connect.Response[queryv1.ResolveSeasonTargetResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ResolveSeasonTarget",
		log.Uint32("season_id", req.Msg.GetSeasonId()),
		log.Uint32("event_no", req.Msg.GetEventNo()),
		log.Uint32("race_no", req.Msg.GetRaceNo()),
		log.Uint32("race_grid_no", req.Msg.GetRaceGridNo()),
	)

	if req.Msg.GetSeasonId() == 0 {
		err := fmt.Errorf("season_id is required")
		l.Error("invalid ResolveSeasonTarget request", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if req.Msg.GetSeasonId() > maxSeasonTargetValue ||
		req.Msg.GetEventNo() > maxSeasonTargetValue ||
		req.Msg.GetRaceNo() > maxSeasonTargetValue ||
		req.Msg.GetRaceGridNo() > maxSeasonTargetValue {
		err := fmt.Errorf("season target values must not exceed %d", maxSeasonTargetValue)
		l.Error("invalid ResolveSeasonTarget request", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, err.Error())
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	target, err := s.resolveSeasonTarget(ctx, req.Msg)
	if err != nil {
		l.Error("failed to resolve season target", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to resolve season target")
		return nil, connect.NewError(s.conversion.MapErrorToRPCCode(err), err)
	}

	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "season target resolved")
	return connect.NewResponse(&queryv1.ResolveSeasonTargetResponse{
		EventId:    uint32(target.EventID),
		RaceId:     uint32(target.RaceID),
		RaceGridId: uint32(target.RaceGridID),
	}), nil
}

// resolveSeasonTarget selects the resolver matching the deepest given parameter.
// Parameters not greater than 0 are considered not given.
//
//nolint:whitespace // editor/linter issue
func (s *service) resolveSeasonTarget(
	ctx context.Context,
	msg *queryv1.ResolveSeasonTargetRequest,
) (*rootrepo.SeasonTarget, error) {
	resolver := s.repo.Queries().QueryResolver()
	seasonID := int32(msg.GetSeasonId())
	eventNo := int32(msg.GetEventNo())
	raceNo := int32(msg.GetRaceNo())
	raceGridNo := int32(msg.GetRaceGridNo())

	switch {
	case eventNo == 0:
		return &rootrepo.SeasonTarget{}, nil
	case raceNo == 0:
		return resolver.ResolveEvent(ctx, seasonID, eventNo)
	case raceGridNo == 0:
		return resolver.ResolveRace(ctx, seasonID, eventNo, raceNo)
	default:
		return resolver.ResolveRaceGrid(ctx, seasonID, eventNo, raceNo, raceGridNo)
	}
}
