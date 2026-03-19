package query

import (
	"context"

	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/conversion"
)

type service struct {
	queryv1connect.UnimplementedQueryServiceHandler
	logger     *log.Logger
	repo       rootrepo.Repository
	txMgr      rootrepo.TransactionManager
	conversion *conversion.Service
}

// New creates the query service handler.
//
//nolint:whitespace // editor/linter issue
func New(
	repo rootrepo.Repository,
	txMgr rootrepo.TransactionManager,
	logger *log.Logger,
) queryv1connect.QueryServiceHandler {
	return &service{
		logger:     logger,
		repo:       repo,
		txMgr:      txMgr,
		conversion: conversion.New(),
	}
}

// ListSimulations returns a list of all simulations.
//
//nolint:whitespace // editor/linter issue
func (s *service) ListSimulations(
	ctx context.Context,
	req *connect.Request[queryv1.ListSimulationsRequest],
) (*connect.Response[queryv1.ListSimulationsResponse], error) {
	l := s.logger.WithCtx(ctx)
	l.Debug("ListSimulations")

	sims, err := s.repo.RacingSims().LoadAll(ctx)
	if err != nil {
		l.Error("failed to load simulation", log.ErrorField(err))
		trace.SpanFromContext(ctx).SetStatus(codes.Error, "failed to load simulations")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	trace.SpanFromContext(ctx).SetStatus(codes.Ok, "simulations loaded")

	return connect.NewResponse(&queryv1.ListSimulationsResponse{
		Items: s.conversion.RacingSimsToSimulations(sims),
	}), nil
}
