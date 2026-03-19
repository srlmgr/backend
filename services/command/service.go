package command

import (
	"context"

	commandv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/command/v1/commandv1connect"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
)

type service struct {
	commandv1connect.UnimplementedCommandServiceHandler
	logger *log.Logger
	repo   rootrepo.Repository
	txMgr  rootrepo.TransactionManager
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
	return &service{logger: logger, repo: repo, txMgr: txMgr}
}

func (s *service) withTx(ctx context.Context, fn func(context.Context) error) error {
	return s.txMgr.RunInTx(ctx, func(txCtx context.Context) error {
		return fn(txCtx)
	})
}

// the concrete service methods are implemented in their respective files,
// e.g. simulation.go, pointsystem.go, etc.
