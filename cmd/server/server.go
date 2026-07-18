package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/authz"
	"github.com/srlmgr/backend/cmd/config"
	"github.com/srlmgr/backend/db/postgres"
	grpcserver "github.com/srlmgr/backend/grpc/server"
	htmlserver "github.com/srlmgr/backend/html/server"
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

	return cmd
}

type server struct {
	ctx        context.Context
	pool       *pgxpool.Pool
	serveErrCh chan error
}

func (s *server) startServers() (err error) {
	s.pool = postgres.InitWithURL(
		config.DBURI,
		postgres.WithTracer(postgres.NewOtlpTracer()))
	defer s.pool.Close()
	s.serveErrCh = make(chan error, 1)
	if config.GRPCEnabled {
		go func() {
			err = s.startGRPC()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to start gRPC server: %v\n", err)
				s.serveErrCh <- err
			}
		}()
	}
	if config.HTMLEnabled {
		go func() {
			err = s.startHTML()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to start HTML server: %v\n", err)
				s.serveErrCh <- err
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

func (s *server) startGRPC() error {
	return grpcserver.Run(s.ctx, s.pool, &grpcserver.Config{
		Address: config.GRPCServerAddress,
		Authn: authn.Config{
			Enabled: config.AuthnEnabled,
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
	return htmlserver.Run(s.ctx, s.pool, &htmlserver.Config{
		Address: config.HTTPServerAddress,
	})
}
