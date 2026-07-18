package admin

import (
	adminv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/admin/v1/adminv1connect"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
)

type service struct {
	adminv1connect.UnimplementedAdminServiceHandler
	logger *log.Logger
	repo   rootrepo.Repository
	txMgr  rootrepo.TransactionManager
}

// New creates the admin service handler.
//
//nolint:whitespace // editor/linter issue
func New(
	repo rootrepo.Repository,
	txMgr rootrepo.TransactionManager,
	logger *log.Logger,
) adminv1connect.AdminServiceHandler {
	return &service{logger: logger, repo: repo, txMgr: txMgr}
}
