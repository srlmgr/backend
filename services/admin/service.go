package admin

import (
	adminv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/admin/v1/adminv1connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/log"
)

type service struct {
	adminv1connect.UnimplementedAdminServiceHandler
	pool   *pgxpool.Pool
	logger *log.Logger
}

// New creates the admin service handler.
func New(pool *pgxpool.Pool, logger *log.Logger) adminv1connect.AdminServiceHandler {
	return &service{pool: pool, logger: logger}
}
