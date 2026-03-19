package server

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

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
			})
		},
	}

	cmd.Flags().StringVar(&config.ServerAddress,
		"listen-address",
		":8080",
		"address to bind the Connect server to")

	return cmd
}
