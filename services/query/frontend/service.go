// Package frontend provides the FrontendService handler for the query API.
package frontend

import (
	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/conversion"
)

type service struct {
	queryv1connect.UnimplementedFrontendServiceHandler
	logger     *log.Logger
	repo       rootrepo.Repository
	conversion *conversion.Service
	tracer     trace.Tracer
}

// New creates the frontend query service handler.
//
//nolint:whitespace // editor/linter issue
func New(
	repo rootrepo.Repository,
	logger *log.Logger,
	tracer trace.Tracer,
) queryv1connect.FrontendServiceHandler {
	return &service{
		logger:     logger,
		repo:       repo,
		conversion: conversion.New(),
		tracer:     tracer,
	}
}
