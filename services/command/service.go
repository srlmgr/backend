package command

import (
	"context"

	commandv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/command/v1/commandv1connect"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/conversion"
)

type service struct {
	commandv1connect.UnimplementedCommandServiceHandler
	logger     *log.Logger
	repo       rootrepo.Repository
	txMgr      rootrepo.TransactionManager
	conversion *conversion.Service
}

var _ commandv1connect.CommandServiceHandler = (*service)(nil)

// New creates the command service handler.
//
//nolint:whitespace //editor/linter issue
func New(
	repo rootrepo.Repository,
	txMgr rootrepo.TransactionManager,
	logger *log.Logger,
) commandv1connect.CommandServiceHandler {
	return &service{
		logger:     logger,
		repo:       repo,
		txMgr:      txMgr,
		conversion: conversion.New(),
	}
}

/*
Note:  the concrete service methods are implemented in their respective files,
e.g. simulation.go, pointsystem.go, etc.
*/

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
