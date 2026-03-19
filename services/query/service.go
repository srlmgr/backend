package query

import (
	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/postgres"
)

type service struct {
	queryv1connect.UnimplementedQueryServiceHandler
	logger *log.Logger
	repo   rootrepo.Repository
}

// New creates the query service handler.
func New(pool *pgxpool.Pool, logger *log.Logger) queryv1connect.QueryServiceHandler {
	return NewWithRepository(postgres.New(pool), logger)
}

// NewWithRepository creates the query service handler with an
// injected repository aggregate.
//
//nolint:whitespace // editor/linter issue
func NewWithRepository(
	repo rootrepo.Repository,
	logger *log.Logger,
) queryv1connect.QueryServiceHandler {
	return &service{logger: logger, repo: repo}
}
