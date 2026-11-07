package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/authz"
	"github.com/srlmgr/backend/cache"
	"github.com/srlmgr/backend/cmd/config"
	"github.com/srlmgr/backend/db/postgres"
	grpcserver "github.com/srlmgr/backend/grpc/server"
	htmlserver "github.com/srlmgr/backend/html/server"
	"github.com/srlmgr/backend/log"
	rootrepo "github.com/srlmgr/backend/repository"
	pgRepos "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/service"
	serviceImpl "github.com/srlmgr/backend/service/impl"
	"github.com/srlmgr/backend/support/iracing/irdata"
)

// NewServerCmd creates the command that runs the Connect-based gRPC server.
func NewServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the Connect gRPC server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(config.DBURI) == "" {
				return fmt.Errorf("--db-uri is required")
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			s := &server{ctx: ctx}
			return s.startServers()
		},
	}

	cmd.Flags().StringVar(&config.HTTPServerAddress,
		"http-address",
		":8080",
		"address to bind the HTTP server to")
	cmd.Flags().StringVar(&config.GRPCServerAddress,
		"grpc-address",
		":9090",
		"address to bind the Connect server to")
	cmd.Flags().BoolVar(&config.GRPCEnabled,
		"grpc-enabled",
		true,
		"enable the Connect server")
	cmd.Flags().BoolVar(&config.HTMLEnabled,
		"html-enabled",
		true,
		"enable the HTML server")
	cmd.Flags().StringVar(&config.HTMLExternalURL,
		"html-external-url",
		"",
		"Use this URL for every navigation link")
	cmd.Flags().StringVar(&config.HTMLContextPart,
		"html-context-part",
		"/vrdb",
		"Context part for HTML server endpoints (used for navigation links)")

	return cmd
}

type server struct {
	ctx      context.Context
	pool     *pgxpool.Pool
	repo     rootrepo.Repository
	service  service.Service
	irClient *iRacingClient
	sessions authn.SessionStore

	serveErrCh chan error
}

//nolint:funlen,gocyclo // coordinates the server lifecycle
func (s *server) startServers() (err error) {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	s.ctx = ctx
	var cacheConfig *cache.Config
	if strings.TrimSpace(config.CacheConfigFile) != "" {
		cacheConfig, err = cache.LoadConfig(config.CacheConfigFile)
		if err != nil {
			return fmt.Errorf("load cache config: %w", err)
		}
	}

	s.pool = postgres.InitWithURL(
		config.DBURI,
		postgres.WithTracer(postgres.NewOtlpTracer()),
	)
	defer s.pool.Close()
	var workers sync.WaitGroup
	defer func() {
		cancel()
		workers.Wait()
		s.closeIRacingClient()
	}()

	repoOpts := []pgRepos.Option{}
	serviceOpts := []serviceImpl.Option{}
	meter := otel.Meter("srlmgr.backend")
	if config.CacheEnabled {
		cm := cache.NewManager()
		if cmErr := cache.InitMetrics(meter); cmErr != nil {
			return fmt.Errorf("init cache metrics: %w", cmErr)
		}
		s.ctx = cache.AddCacheConfigToContext(s.ctx, cm, cacheConfig)
		repoOpts = append(repoOpts, pgRepos.WithCache(s.ctx, cacheConfig, cm))
		serviceOpts = append(serviceOpts, serviceImpl.WithCache(s.ctx, cacheConfig, cm))
	}
	s.repo = pgRepos.New(s.pool, repoOpts...)
	if config.AuthnPersistentSessionsEnabled && config.GRPCEnabled {
		checkCtx, checkCancel := context.WithTimeout(s.ctx, 10*time.Second)
		store, storeErr := authn.NewPostgresSessionStore(
			checkCtx, s.repo.AuthSessions(), rootrepo.NewBobTransactionFromPool(s.pool),
		)
		checkCancel()
		if storeErr != nil {
			return fmt.Errorf("initialize persistent sessions: %w", storeErr)
		}
		s.sessions = store
		workers.Add(1)
		go func() {
			defer workers.Done()
			cleanupSessions(ctx, store)
		}()
	}
	s.service = serviceImpl.New(
		s.repo,
		log.GetFromContext(s.ctx).Named("service"),
		serviceOpts...,
	)
	if config.IRacingCfg.Enabled {
		var iracingErr error
		if iracingErr = s.initIRacingClient(); iracingErr != nil {
			log.Warn("failed to initialize iRacing client", log.ErrorField(iracingErr))
		}
	}
	s.serveErrCh = make(chan error, 2)
	if config.GRPCEnabled {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if serveErr := s.startGRPC(); serveErr != nil {
				fmt.Fprintf(os.Stderr, "failed to start gRPC server: %v\n", serveErr)
				s.serveErrCh <- serveErr
			}
		}()
	}
	if config.HTMLEnabled {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if serveErr := s.startHTML(); serveErr != nil {
				fmt.Fprintf(os.Stderr, "failed to start HTML server: %v\n", serveErr)
				s.serveErrCh <- serveErr
			}
		}()
	}
	select {
	case serveErr := <-s.serveErrCh:
		if serveErr != nil {
			return fmt.Errorf("server reported error: %w", serveErr)
		}
		return nil
	case <-s.ctx.Done():
		fmt.Fprintf(os.Stderr, "context was closed\n")
	}
	return nil
}

func cleanupSessions(ctx context.Context, store *authn.PostgresSessionStore) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := store.CleanupExpired(cleanupCtx, 1000)
		cancel()
		if err != nil && ctx.Err() == nil {
			log.Error("failed to clean up expired sessions", log.ErrorField(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// initialize the iRacing client and add it to the server context
func (s *server) initIRacingClient() error {
	irClient, initErr := InitIRacingClient()
	if initErr != nil {
		return initErr
	}
	s.irClient = irClient
	s.ctx = irdata.AddToContext(s.ctx, s.irClient.API)
	return nil
}

func (s *server) closeIRacingClient() {
	if s.irClient != nil {
		s.irClient.Close()
		s.irClient = nil
	}
}

func (s *server) startGRPC() error {
	return grpcserver.Run(s.ctx, s.pool, &grpcserver.Config{
		Address:  config.GRPCServerAddress,
		Repo:     s.repo,
		Services: s.service,
		Authn: authn.Config{
			Enabled: config.AuthnEnabled,
			Store:   s.sessions,
			IDP: authn.IDPConfig{
				Enabled:        config.IDPEnabled,
				IssuerURL:      config.IDPIssuerURL,
				ClientID:       config.IDPClientID,
				ClientSecret:   config.IDPClientSecret,
				CallbackURL:    config.IDPCallbackURL,
				FrontendURL:    config.IDPFrontendURL,
				RefreshSkew:    config.IDPRefreshSkew,
				CookieSecure:   true,
				CookieHTTPOnly: true,
			},
			APIToken: authn.APITokenConfig{
				FilePath:        config.AuthnAPITokenFilePath,
				RefreshInterval: config.AuthnAPITokenRefreshWindow,
			},
		},
		Authz: authz.Config{
			Enabled:          config.AuthzEnabled,
			PolicyPath:       config.AuthzPolicyPath,
			DecisionCacheTTL: config.AuthzDecisionCacheTTL,
		},
	})
}

func (s *server) startHTML() error {
	return htmlserver.Run(s.ctx, &htmlserver.Config{
		Address:     config.HTTPServerAddress,
		ExternalURL: config.HTMLExternalURL,
		ContextPart: config.HTMLContextPart,
		Repo:        s.repo,
		Services:    s.service,
	})
}
