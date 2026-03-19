package importsvc

import (
	importv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/import/v1/importv1connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/log"
)

type service struct {
	importv1connect.UnimplementedImportServiceHandler
	pool   *pgxpool.Pool
	logger *log.Logger
}

// New creates the import service handler.
func New(pool *pgxpool.Pool, logger *log.Logger) importv1connect.ImportServiceHandler {
	return &service{pool: pool, logger: logger}
}
