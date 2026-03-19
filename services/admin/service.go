package admin

import (
	adminv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/admin/v1/adminv1connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/postgres"
)

type service struct {
	adminv1connect.UnimplementedAdminServiceHandler

	logger *log.Logger
	repo   rootrepo.Repository
}

// New creates the admin service handler.
func New(pool *pgxpool.Pool, logger *log.Logger) adminv1connect.AdminServiceHandler {
	return NewWithRepository(postgres.New(pool), logger)
}

// NewWithRepository creates the admin service handler with an
// injected repository aggregate.
//
//nolint:whitespace // editor/linter issue
func NewWithRepository(
	repo rootrepo.Repository,
	logger *log.Logger,
) adminv1connect.AdminServiceHandler {
	return &service{logger: logger, repo: repo}
}
