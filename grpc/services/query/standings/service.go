// Package standings provides the StandingsService handler for the query API.
package standings

import (
	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/grpc/services/conversion"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/service"
)

type standingsService struct {
	queryv1connect.UnimplementedStandingsServiceHandler
	logger     *log.Logger
	repo       rootrepo.Repository
	conversion *conversion.ConvService
	tracer     trace.Tracer
	svc        service.Service
}

// New creates the standings query service handler.
//
//nolint:whitespace // editor/linter issue
func New(
	repo rootrepo.Repository,
	svc service.Service,
	logger *log.Logger,
	tracer trace.Tracer,
) queryv1connect.StandingsServiceHandler {
	return &standingsService{
		logger:     logger,
		repo:       repo,
		conversion: conversion.New(),
		tracer:     tracer,
		svc:        svc,
	}
}
