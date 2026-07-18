// Package bookings provides the BoookingsService handler for the query API.
package bookings

import (
	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/grpc/services/conversion"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
)

type service struct {
	queryv1connect.UnimplementedBookingsServiceHandler
	logger     *log.Logger
	repo       rootrepo.Repository
	conversion *conversion.Service
	tracer     trace.Tracer
}

// New creates the bookings query service handler.
//
//nolint:whitespace // editor/linter issue
func New(
	repo rootrepo.Repository,
	logger *log.Logger,
	tracer trace.Tracer,
) queryv1connect.BookingsServiceHandler {
	return &service{
		logger:     logger,
		repo:       repo,
		conversion: conversion.New(),
		tracer:     tracer,
	}
}
