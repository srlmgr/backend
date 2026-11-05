package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	adminv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/v2/backend/admin/v1/adminv1connect"
	commandv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/v2/backend/command/v1/commandv1connect"
	importv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/v2/backend/import/v1/importv1connect"
	queryv1connect "buf.build/gen/go/srlmgr/api/connectrpc/go/v2/backend/query/v1/queryv1connect"
	connect "connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/grpchealth/v2"
	"connectrpc.com/grpcreflect/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/authz"
	adminservice "github.com/srlmgr/backend/grpc/services/admin"
	commandservice "github.com/srlmgr/backend/grpc/services/command"
	importservice "github.com/srlmgr/backend/grpc/services/importsvc"
	queryservice "github.com/srlmgr/backend/grpc/services/query"
	bookingservice "github.com/srlmgr/backend/grpc/services/query/bookings"
	frontendservice "github.com/srlmgr/backend/grpc/services/query/frontend"
	standingsservice "github.com/srlmgr/backend/grpc/services/query/standings"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/service"
)

const shutdownTimeout = 10 * time.Second

// Config configures the Connect server.
type Config struct {
	Address  string
	Authn    authn.Config
	Authz    authz.Config
	Repo     repository.Repository // shared repository instance, reused across servers
	Services service.Service       // shared service instance, reused across servers
}

// Run starts the Connect server and blocks until shutdown completes.
func Run(ctx context.Context, pool *pgxpool.Pool, cfg *Config) error {
	logger := getLogger(ctx)

	if err := validateSecurityConfig(cfg); err != nil {
		return err
	}

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

//nolint:whitespace,funlen //editor/linter issue
func newHTTPServer(
	ctx context.Context,
	cfg *Config,
	pool *pgxpool.Pool,
	logger *log.Logger,
) (*http.Server, error) {
	authnManager, err := authn.NewManager(ctx, &cfg.Authn, logger)
	if err != nil {
		return nil, fmt.Errorf("create authentication manager: %w", err)
	}

	interceptors, err := newConnectInterceptors(
		ctx,
		cfg,
		pool,
		logger,
		authnManager.Interceptor(),
	)
	if err != nil {
		return nil, err
	}

	httpCapabilityAuthorizer, err := authz.NewCapabilityAuthorizer(ctx, cfg.Authz)
	if err != nil {
		return nil, fmt.Errorf("create http authorization evaluator: %w", err)
	}

	txManager := repository.NewBobTransactionFromPool(pool)
	repo := cfg.Repo
	importHandler := importservice.New(repo, txManager, logger.Named("services.import"))

	mux := http.NewServeMux()
	authnManager.RegisterHTTPHandlers(mux)
	registerMultipartUploadHandler(
		mux,
		logger,
		authnManager,
		httpCapabilityAuthorizer,
		repo,
		importHandler,
	)
	apiServer := registerConnectHandlers(
		mux, txManager, repo, cfg.Services, logger, interceptors...,
	)
	registerHealthAndReflection(mux, apiServer)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return &http.Server{
		Addr:              cfg.Address,
		Handler:           otelhttp.NewHandler(newCORS().Handler(mux), "backend-server"),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}, nil
}

func newCORS() *cors.Cors {
	return cors.New(cors.Options{
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
		},
		AllowOriginFunc: func(origin string) bool {
			// Allow all origins, which effectively disables CORS.
			return true
		},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		ExposedHeaders: []string{
			// Content-Type is in the default safelist.
			"Accept",
			"Accept-Encoding",
			"Accept-Post",
			"Connect-Accept-Encoding",
			"Connect-Content-Encoding",
			"Content-Encoding",
			"Grpc-Accept-Encoding",
			"Grpc-Encoding",
			"Grpc-Message",
			"Grpc-Status",
			"Grpc-Status-Details-Bin",
		},
		// Let browsers cache CORS information for longer, which reduces the number
		// of preflight requests. Any changes to ExposedHeaders won't take effect
		// until the cached data expires. FF caps this value at 24h, and modern
		// Chrome caps it at 2h.
		MaxAge: int(2 * time.Hour / time.Second),
	})
}

func validateSecurityConfig(cfg *Config) error {
	if !cfg.Authn.Enabled {
		return nil
	}

	apiTokenEnabled := strings.TrimSpace(cfg.Authn.APIToken.FilePath) != ""
	idpEnabled := cfg.Authn.IDP.Enabled
	if !idpEnabled && !apiTokenEnabled {
		return fmt.Errorf(
			"authentication is enabled but both idp and api-token validators are disabled",
		)
	}

	if idpEnabled {
		if strings.TrimSpace(cfg.Authn.IDP.IssuerURL) == "" ||
			strings.TrimSpace(cfg.Authn.IDP.ClientID) == "" ||
			strings.TrimSpace(cfg.Authn.IDP.ClientSecret) == "" ||
			strings.TrimSpace(cfg.Authn.IDP.CallbackURL) == "" {

			return fmt.Errorf(
				"idp authn requires issuer-url, client-id, client-secret and callback-url",
			)
		}

		if strings.TrimSpace(cfg.Authn.IDP.FrontendURL) == "" {
			return fmt.Errorf("idp authn requires frontend-url for post-login redirects")
		}
	}

	return nil
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
	txManager repository.TransactionManager,
	repo repository.Repository,
	svc service.Service,
	logger *log.Logger,
	interceptors ...connect.ServerInterceptor,
) *connect.Server {
	tracer := otel.Tracer("backend.grpc")
	server := connect.NewServer(interceptors...)
	adminv1connect.RegisterAdminServiceHandler(
		server,
		adminservice.New(repo, txManager, logger.Named("services.admin")),
	)
	commandv1connect.RegisterCommandServiceHandler(
		server,
		commandservice.New(repo, txManager, logger.Named("services.command")),
	)
	importv1connect.RegisterImportServiceHandler(
		server,
		importservice.New(repo, txManager, logger.Named("services.import")),
	)
	queryv1connect.RegisterQueryServiceHandler(
		server,
		queryservice.New(repo, svc, txManager, logger.Named("services.query"), tracer),
	)
	queryv1connect.RegisterFrontendServiceHandler(
		server,
		frontendservice.New(repo, logger.Named("services.frontend"), tracer),
	)
	queryv1connect.RegisterBookingsServiceHandler(
		server,
		bookingservice.New(repo, logger.Named("services.bookings"), tracer),
	)
	queryv1connect.RegisterStandingsServiceHandler(
		server,
		standingsservice.New(repo, svc, logger.Named("services.standings"), tracer),
	)

	// v2 limits reads to 4 MiB per message by default; v1 was unbounded.
	// Keep the v1 behavior since UploadResultsFile carries file payloads.
	connecthttp.Mount(mux, server, connecthttp.WithReadMaxBytes(0))
	return server
}

// registerHealthAndReflection serves gRPC health and reflection on a separate
// connect server so they bypass the authn/authz interceptors (as in v1).
// Reflection lists the services of both servers.
func registerHealthAndReflection(mux *http.ServeMux, apiServer *connect.Server) {
	server := connect.NewServer()
	grpchealth.Register(server, grpchealth.NewStaticChecker())
	grpcreflect.Register(server, grpcreflect.WithNamer(grpcreflect.NamerFunc(
		func() []string { return serviceNames(apiServer, server) },
	)))
	connecthttp.Mount(mux, server)
}

func serviceNames(servers ...*connect.Server) []string {
	var names []string
	seen := map[string]struct{}{}
	for _, srv := range servers {
		for spec := range srv.Specs() {
			name, _, ok := strings.Cut(strings.TrimPrefix(spec.Procedure, "/"), "/")
			if !ok {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	return names
}
