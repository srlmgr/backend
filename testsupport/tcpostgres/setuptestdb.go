package tcpostgres

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/srlmgr/backend/db/migrate"
)

// SetupTestDB starts (or reuses) the shared testcontainers postgres instance and
// makes sure the template database used for per-test/per-package cloning exists.
// It returns the admin connection URL (targeting the container's "postgres"
// maintenance database), not a ready-to-use working pool.
func SetupTestDB() (string, error) {
	ctx := context.Background()
	port, err := nat.NewPort("tcp", "5432")
	if err != nil {
		log.Fatal(err)
	}
	container, err := SetupPostgres(
		ctx,
		WithPort(port.Port()),
		WithInitialDatabase("postgres", "password", "postgres"),
		WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
		WithName("srlmgr-test"),
	)
	if err != nil {
		return "", err
	}

	containerPort, _ := container.MappedPort(ctx, port.Port())
	host, _ := container.Host(ctx)
	adminURL := fmt.Sprintf("postgresql://postgres:password@%s:%s/postgres",
		host, containerPort.Port())

	if err := EnsureTemplateDB(ctx, adminURL); err != nil {
		return "", err
	}

	return adminURL, nil
}

// SetupExternalTestDB migrates and returns the externally provided TESTDB_URL,
// used as-is (no template/clone support for this flow).
func SetupExternalTestDB() (*pgxpool.Pool, error) {
	dbURL := os.Getenv("TESTDB_URL")
	if err := migrate.MigrateDB(dbURL); err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	return pool, nil
}

func clearTables(pool *pgxpool.Pool, tables []string) error {
	err := pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(
			context.Background(),
			fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY ",
				strings.Join(tables, ", ")),
		); err != nil {
			return err
		}
		return nil
	})
	return err
}

func ClearAllTables(pool *pgxpool.Pool) error {
	tables := []string{
		"booking_entries",
		"result_entries",
		"event_processing_audit",
		"import_batches",
		"event_team_standings",
		"event_driver_standings",
		"season_car_classes",
		"season_car_model_variants",
		"season_team_standings",
		"race_grids",
		"races",
		"season_drivers",
		"team_drivers",
		"events",
		"season_driver_standings",
		"teams",
		"car_classes_to_car_models",
		"simulation_car_aliases",
		"seasons",
		"simulation_track_layout_aliases",
		"car_model_variants",
		"car_models",
		"simulation_driver_aliases",
		"series",
		"track_layouts",
		"point_rules",
		"racing_sims",
		"point_systems",
		"tracks",
		"car_manufacturers",
		"car_classes",
		"drivers",
	}
	err := clearTables(pool, tables)
	return err
}
