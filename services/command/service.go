package command

import (
	commandv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/command/v1/commandv1connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/log"
)

type service struct {
	commandv1connect.UnimplementedCommandServiceHandler
	pool   *pgxpool.Pool
	logger *log.Logger
}

// New creates the command service handler.
//
//nolint:whitespace //editor/linter issue
func New(
	pool *pgxpool.Pool,
	logger *log.Logger,
) commandv1connect.CommandServiceHandler {
	return &service{pool: pool, logger: logger}
}
