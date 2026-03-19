package command

import (
	commandv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/command/v1/commandv1connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/postgres"
)

type service struct {
	commandv1connect.UnimplementedCommandServiceHandler
	logger *log.Logger
	repo   rootrepo.Repository
}

// New creates the command service handler.
//
//nolint:whitespace //editor/linter issue
func New(
	pool *pgxpool.Pool,
	logger *log.Logger,
) commandv1connect.CommandServiceHandler {
	return NewWithRepository(postgres.New(pool), logger)
}

// NewWithRepository creates the command service handler with an
// injected repository aggregate.
//
//nolint:whitespace // editor/linter issue
func NewWithRepository(
	repo rootrepo.Repository,
	logger *log.Logger,
) commandv1connect.CommandServiceHandler {
	return &service{logger: logger, repo: repo}
}
