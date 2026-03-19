package importsvc

import (
	importv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/import/v1/importv1connect"

	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
)

type service struct {
	importv1connect.UnimplementedImportServiceHandler

	logger *log.Logger
	repo   rootrepo.Repository
	txMgr  rootrepo.TransactionManager
}

// New creates the import service handler.
//
//nolint:whitespace // editor/linter issue
func New(
	repo rootrepo.Repository,
	txMgr rootrepo.TransactionManager,
	logger *log.Logger,
) importv1connect.ImportServiceHandler {
	return &service{logger: logger, repo: repo, txMgr: txMgr}
}
