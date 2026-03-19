package server

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/cmd/config"
	backendserver "github.com/srlmgr/backend/server"
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

			return backendserver.Run(ctx, backendserver.Config{
				Address: config.ServerAddress,
				DBURI:   config.DBURI,
				AuthnConfig: authn.Config{
					JWT: authn.JWTConfig{
						Issuer:               config.JWTIssuer,
						Audience:             config.JWTAudience,
						JWKSUrl:              config.JWKSUrl,
						CacheRefreshInterval: config.JWKSCacheRefreshInterval,
						ClockSkew:            config.JWTClockSkew,
					},
					APIToken: authn.APITokenConfig{
						FilePath:       config.APITokenFilePath,
						ReloadInterval: config.APITokenReloadInterval,
					},
				},
			})
		},
	}

	cmd.Flags().StringVar(&config.ServerAddress,
		"listen-address",
		":8080",
		"address to bind the Connect server to")

	cmd.Flags().StringVar(&config.JWTIssuer,
		"jwt-issuer",
		"",
		"expected JWT issuer (iss claim)")
	cmd.Flags().StringVar(&config.JWTAudience,
		"jwt-audience",
		"",
		"expected JWT audience (aud claim)")
	cmd.Flags().StringVar(&config.JWKSUrl,
		"jwks-url",
		"",
		"URL of the JWKS endpoint for JWT signature verification")
	cmd.Flags().DurationVar(&config.JWKSCacheRefreshInterval,
		"jwks-cache-refresh",
		5*time.Minute,
		"how often to refresh the JWKS key cache")
	cmd.Flags().DurationVar(&config.JWTClockSkew,
		"jwt-clock-skew",
		5*time.Second,
		"allowed clock skew for JWT time-based claim validation")
	cmd.Flags().StringVar(&config.APITokenFilePath,
		"api-token-file",
		"",
		"path to the API token store YAML file")
	cmd.Flags().DurationVar(&config.APITokenReloadInterval,
		"api-token-reload-interval",
		0,
		"how often to reload the API token file (0 = startup only)")

	return cmd
}
