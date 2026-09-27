package tcpostgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/srlmgr/backend/db/migrate"
)

const (
	// TemplateDBName is the pre-migrated database every test/package clone is copied from.
	TemplateDBName = "srlmgr_template"
	// StaleDBPrefix marks databases cloned from the template so they can be swept later.
	StaleDBPrefix = "srlmgr_test_"
	// staleDBAge is how long an orphaned clone survives before the sweep drops it.
	staleDBAge = 30 * time.Minute
	// advisoryLockKey serializes template creation/sweeping across concurrent binaries.
	advisoryLockKey = 872465017
)

// EnsureTemplateDB creates and migrates the shared template database if it doesn't
// exist yet, and opportunistically drops stale clones left behind by crashed runs.
// Safe to call concurrently from multiple processes sharing the same Postgres instance.
func EnsureTemplateDB(ctx context.Context, adminURL string) error {
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return fmt.Errorf("connect admin db: %w", err)
	}
	defer conn.Close(ctx)

	lockQuery := "SELECT pg_advisory_lock($1)"
	if _, err := conn.Exec(ctx, lockQuery, advisoryLockKey); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockKey)
	}()

	var exists bool
	if err := conn.QueryRow(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", TemplateDBName,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check template db existence: %w", err)
	}

	if !exists {
		createStmt := fmt.Sprintf(
			"CREATE DATABASE %s", pgx.Identifier{TemplateDBName}.Sanitize(),
		)
		if _, err := conn.Exec(ctx, createStmt); err != nil {
			return fmt.Errorf("create template db: %w", err)
		}

		templateURL, err := withDatabase(adminURL, TemplateDBName)
		if err != nil {
			return fmt.Errorf("build template db url: %w", err)
		}
		if err := migrate.MigrateDB(templateURL); err != nil {
			return fmt.Errorf("migrate template db: %w", err)
		}
	}

	return sweepStaleDatabases(ctx, conn)
}

// CloneDatabase creates a new database from the template and returns its name and
// connection URL (derived from adminURL). The caller is responsible for dropping it
// via DropDatabase once done.
//
//nolint:whitespace // multiline signature style
func CloneDatabase(
	ctx context.Context, adminURL string,
) (dbName, dbURL string, err error) {
	dbName, err = newCloneName()
	if err != nil {
		return "", "", err
	}

	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return "", "", fmt.Errorf("connect admin db: %w", err)
	}
	defer conn.Close(ctx)

	createStmt := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s",
		pgx.Identifier{dbName}.Sanitize(), pgx.Identifier{TemplateDBName}.Sanitize())
	if _, execErr := conn.Exec(ctx, createStmt); execErr != nil {
		return "", "", fmt.Errorf("clone template db: %w", execErr)
	}

	dbURL, err = withDatabase(adminURL, dbName)
	if err != nil {
		return "", "", fmt.Errorf("build clone db url: %w", err)
	}

	return dbName, dbURL, nil
}

// DropDatabase force-drops a database (terminating any lingering connections first).
func DropDatabase(ctx context.Context, adminURL, dbName string) error {
	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return fmt.Errorf("connect admin db: %w", err)
	}
	defer conn.Close(ctx)

	dropStmt := fmt.Sprintf(
		"DROP DATABASE IF EXISTS %s WITH (FORCE)",
		pgx.Identifier{dbName}.Sanitize(),
	)
	if _, err := conn.Exec(ctx, dropStmt); err != nil {
		return fmt.Errorf("drop db %s: %w", dbName, err)
	}
	return nil
}

// sweepStaleDatabases best-effort drops clones older than staleDBAge with no active
// connections, cleaning up after crashed/panicked test runs. Must be called while
// holding the advisory lock so it doesn't race a sibling process mid-clone.
func sweepStaleDatabases(ctx context.Context, conn *pgx.Conn) error {
	rows, err := conn.Query(ctx,
		"SELECT datname FROM pg_database WHERE datname LIKE $1", StaleDBPrefix+"%")
	if err != nil {
		return fmt.Errorf("list stale candidate dbs: %w", err)
	}

	var candidates []string
	for rows.Next() {
		var datname string
		if err := rows.Scan(&datname); err != nil {
			rows.Close()
			return fmt.Errorf("scan stale candidate db: %w", err)
		}
		candidates = append(candidates, datname)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate stale candidate dbs: %w", err)
	}

	for _, datname := range candidates {
		createdAt, ok := parseCloneTimestamp(datname)
		if !ok || time.Since(createdAt) < staleDBAge {
			continue
		}

		var activeConns int
		if err := conn.QueryRow(
			ctx,
			"SELECT count(*) FROM pg_stat_activity WHERE datname = $1", datname,
		).Scan(&activeConns); err != nil {
			continue // best-effort: skip on error, try again next run
		}
		if activeConns > 0 {
			continue
		}

		dropStmt := fmt.Sprintf(
			"DROP DATABASE IF EXISTS %s WITH (FORCE)",
			pgx.Identifier{datname}.Sanitize(),
		)
		_, _ = conn.Exec(ctx, dropStmt) // best-effort cleanup
	}

	return nil
}

func newCloneName() (string, error) {
	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("generate clone db suffix: %w", err)
	}
	return fmt.Sprintf(
		"%s%d_%s",
		StaleDBPrefix,
		time.Now().UnixMilli(),
		hex.EncodeToString(suffix),
	), nil
}

// parseCloneTimestamp extracts the creation time embedded in a clone db name
// (srlmgr_test_<unixmillis>_<rand>).
func parseCloneTimestamp(dbName string) (time.Time, bool) {
	rest, ok := strings.CutPrefix(dbName, StaleDBPrefix)
	if !ok {
		return time.Time{}, false
	}
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) == 0 {
		return time.Time{}, false
	}
	millis, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.UnixMilli(millis), true
}

func withDatabase(rawURL, dbName string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	u.Path = "/" + dbName
	return u.String(), nil
}
