package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	adminv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/admin/v1/adminv1connect"
	commandv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/command/v1/commandv1connect"
	importv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/import/v1/importv1connect"
	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/backend/query/v1/queryv1connect"
	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/srlmgr/backend/db/postgres"
	"github.com/srlmgr/backend/log"
	adminservice "github.com/srlmgr/backend/services/admin"
	commandservice "github.com/srlmgr/backend/services/command"
	importservice "github.com/srlmgr/backend/services/importsvc"
	queryservice "github.com/srlmgr/backend/services/query"
)

const shutdownTimeout = 10 * time.Second

// Config configures the Connect server.
type Config struct {
	Address string
	DBURI   string
}

// Run starts the Connect server and blocks until shutdown completes.
func Run(ctx context.Context, cfg Config) error {
	logger := getLogger(ctx)

	pool := postgres.InitWithURL(cfg.DBURI, postgres.WithTracer(postgres.NewOtlpTracer()))
	defer pool.Close()

	httpServer, err := newHTTPServer(ctx, cfg, pool, logger)
	if err != nil {
		return err
	}

	return serveHTTPServer(ctx, httpServer, cfg.Address, logger)
}

func getLogger(ctx context.Context) *log.Logger {
	logger := log.GetFromContext(ctx)
	if logger != nil {
		return logger
	}

	return log.Default()
}

//nolint:whitespace //editor/linter issue
func newHTTPServer(
	ctx context.Context,
	cfg Config,
	pool *pgxpool.Pool,
	logger *log.Logger,
) (*http.Server, error) {
	handlerOptions, err := newConnectHandlerOptions(logger)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	registerConnectHandlers(mux, pool, logger, handlerOptions...)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return &http.Server{
		Addr:              cfg.Address,
		Handler:           otelhttp.NewHandler(mux, "backend-server"),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}, nil
}

//nolint:whitespace //editor/linter issue
func serveHTTPServer(
	ctx context.Context,
	httpServer *http.Server,
	address string,
	logger *log.Logger,
) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}

	logger.Info("starting connect server", log.String("address", listener.Addr().String()))

	serveErrCh := make(chan error, 1)
	go func() {
		defer close(serveErrCh)
		if serveErr := httpServer.Serve(listener); serveErr != nil &&
			!errors.Is(serveErr, http.ErrServerClosed) {

			serveErrCh <- serveErr
		}
	}()

	select {
	case serveErr := <-serveErrCh:
		if serveErr != nil {
			return fmt.Errorf("serve connect server: %w", serveErr)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutting down connect server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown connect server: %w", err)
	}

	if serveErr := <-serveErrCh; serveErr != nil {
		return fmt.Errorf("serve connect server: %w", serveErr)
	}

	return nil
}

//nolint:whitespace //editor/linter issue
func registerConnectHandlers(
	mux *http.ServeMux,
	pool *pgxpool.Pool,
	logger *log.Logger,
	opts ...connect.HandlerOption,
) {
	adminPath, adminHandler := adminv1connect.NewAdminServiceHandler(
		adminservice.New(pool, logger.Named("services.admin")),
		opts...,
	)
	commandPath, commandHandler := commandv1connect.NewCommandServiceHandler(
		commandservice.New(pool, logger.Named("services.command")),
		opts...,
	)
	importPath, importHandler := importv1connect.NewImportServiceHandler(
		importservice.New(pool, logger.Named("services.import")),
		opts...,
	)
	queryPath, queryHandler := queryv1connect.NewQueryServiceHandler(
		queryservice.New(pool, logger.Named("services.query")),
		opts...,
	)

	mux.Handle(adminPath, adminHandler)
	mux.Handle(commandPath, commandHandler)
	mux.Handle(importPath, importHandler)
	mux.Handle(queryPath, queryHandler)
}
