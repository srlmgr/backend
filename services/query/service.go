package query

import (
	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect"
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
	tracer     trace.Tracer
}

// New creates the query service handler.
//
//nolint:whitespace // editor/linter issue
func New(
	repo rootrepo.Repository,
	txMgr rootrepo.TransactionManager,
	logger *log.Logger,
	tracer trace.Tracer,
) queryv1connect.QueryServiceHandler {
	return &service{
		logger:     logger,
		repo:       repo,
		txMgr:      txMgr,
		tracer:     tracer,
		conversion: conversion.New(),
	}
}

/*
Note:  the concrete service methods are implemented in their respective files,
e.g. simulation.go, pointsystem.go, etc.
*/
