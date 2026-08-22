package query

import (
	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect"
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/grpc/services/conversion"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	gs "github.com/srlmgr/backend/service"
)

type service struct {
	queryv1connect.UnimplementedQueryServiceHandler
	logger     *log.Logger
	repo       rootrepo.Repository
	svc        gs.Service
	txMgr      rootrepo.TransactionManager
	conversion *conversion.ConvService
	tracer     trace.Tracer
}

// New creates the query service handler.
//
//nolint:whitespace // editor/linter issue
func New(
	repo rootrepo.Repository,
	svc gs.Service,
	txMgr rootrepo.TransactionManager,
	logger *log.Logger,
	tracer trace.Tracer,
) queryv1connect.QueryServiceHandler {
	return &service{
		logger:     logger,
		repo:       repo,
		svc:        svc,
		txMgr:      txMgr,
		tracer:     tracer,
		conversion: conversion.New(),
	}
}

/*
Note:  the concrete service methods are implemented in their respective files,
e.g. simulation.go, pointsystem.go, etc.
*/
