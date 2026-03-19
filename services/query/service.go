package query

import (
	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/log"
)

type service struct {
	queryv1connect.UnimplementedQueryServiceHandler
	pool   *pgxpool.Pool
	logger *log.Logger
}

// New creates the query service handler.
func New(pool *pgxpool.Pool, logger *log.Logger) queryv1connect.QueryServiceHandler {
	return &service{pool: pool, logger: logger}
}
