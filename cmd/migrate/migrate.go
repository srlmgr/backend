package migrate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srlmgr/backend/cmd/config"
	dbmigrate "github.com/srlmgr/backend/db/migrate"
	"github.com/srlmgr/backend/log"
)

// NewMigrateCmd creates the database migration command.
func NewMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(config.DBURI) == "" {
				return errors.New("db-uri is required")
			}

			if err := dbmigrate.MigrateDB(config.DBURI); err != nil {
				return fmt.Errorf("run database migrations: %w", err)
			}

			log.GetFromContext(cmd.Context()).Info("database migrations completed")
			return nil
		},
	}

	return cmd
}
