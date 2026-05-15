package server

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/authz"
	"github.com/srlmgr/backend/cmd/config"
	backendserver "github.com/srlmgr/backend/server"
)

// NewServerCmd creates the command that runs the Connect-based gRPC server.
//
//nolint:funlen // command wiring keeps all flag/config mapping local
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

			return backendserver.Run(ctx, &backendserver.Config{
				Address: config.ServerAddress,
				DBURI:   config.DBURI,
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
		},
	}

	cmd.Flags().StringVar(&config.ServerAddress,
		"listen-address",
		":8080",
		"address to bind the Connect server to")

	return cmd
}
