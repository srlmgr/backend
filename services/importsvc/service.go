package importsvc

import (
	importv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/import/v1/importv1connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/postgres"
)

type service struct {
	importv1connect.UnimplementedImportServiceHandler

	logger *log.Logger
	repo   rootrepo.Repository
}

// New creates the import service handler.
func New(pool *pgxpool.Pool, logger *log.Logger) importv1connect.ImportServiceHandler {
	return NewWithRepository(postgres.New(pool), logger)
}

// NewWithRepository creates the import service handler with an
// injected repository aggregate.
//
//nolint:whitespace // editor/linter issue
func NewWithRepository(
	repo rootrepo.Repository,
	logger *log.Logger,
) importv1connect.ImportServiceHandler {
	return &service{logger: logger, repo: repo}
}
