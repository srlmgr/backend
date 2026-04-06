package importsvc

import (
	"context"

	importv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/import/v1/importv1connect"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/services/importsvc/importer"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

type service struct {
	importv1connect.UnimplementedImportServiceHandler

	logger     *log.Logger
	repo       rootrepo.Repository
	txMgr      rootrepo.TransactionManager
	conversion *conversion.Service
	processor  *importer.Factory
}

var _ importv1connect.ImportServiceHandler = (*service)(nil)

// New creates the import service handler.
//
//nolint:whitespace // editor/linter issue
func New(
	repo rootrepo.Repository,
	txMgr rootrepo.TransactionManager,
	logger *log.Logger,
) importv1connect.ImportServiceHandler {
	return &service{
		logger:     logger,
		repo:       repo,
		txMgr:      txMgr,
		conversion: conversion.New(),
		processor:  importer.NewDefaultFactory(),
	}
}

func (s *service) withTx(ctx context.Context, fn func(context.Context) error) error {
	return s.txMgr.RunInTx(ctx, func(txCtx context.Context) error {
		return fn(txCtx)
	})
}

func (s *service) principal(ctx context.Context) *authn.Principal {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		s.logger.WithCtx(ctx).Warn("principal not found in context")
		return nil
	}
	return &principal
}

func (s *service) execUser(ctx context.Context) string {
	principal := s.principal(ctx)
	if principal == nil {
		return "system"
	}
	return principal.Name
}

//nolint:whitespace // editor/linter issue
func (s *service) resolveProcessorForEvent(
	ctx context.Context,
	epi *processor.EventProcInfo,
) (importer.ProcessImport, *models.RacingSim, error) {
	simulation, err := s.repo.RacingSims().LoadByID(ctx, epi.Series.SimulationID)
	if err != nil {
		return nil, nil, err
	}

	importProcessor, err := s.processor.Get(simulation.Name)
	if err != nil {
		return nil, nil, err
	}

	return importProcessor, simulation, nil
}
