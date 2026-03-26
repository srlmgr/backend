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
	database "github.com/srlmgr/backend/db/postgres"
)

// create a pg connection pool for the srlmgr testdatabase
func SetupTestDB() (*pgxpool.Pool, error) {
	ctx := context.Background()
	port, err := nat.NewPort("tcp", "5432")
	if err != nil {
		log.Fatal(err)
	}
	container, err := SetupPostgres(ctx,
		WithPort(port.Port()),
		WithInitialDatabase("postgres", "password", "postgres"),
		WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
		WithName("srlmgr-test"),
	)
	if err != nil {
		return nil, err
	}

	containerPort, _ := container.MappedPort(ctx, port)
	host, _ := container.Host(ctx)
	dbURL := fmt.Sprintf("postgresql://postgres:password@%s:%s/postgres",
		host, containerPort.Port())

	err = migrate.MigrateDB(dbURL)
	if err != nil {
		return nil, err
	}

	pool := database.InitWithURL(dbURL)
	return pool, nil
}

// create a pg connection pool for the local iracelog testdatabase
func SetupExternalTestDB() (*pgxpool.Pool, error) {
	dbURL := os.Getenv("TESTDB_URL")
	if err := migrate.MigrateDB(dbURL); err != nil {
		return nil, err
	}

	pool := database.InitWithURL(dbURL)
	return pool, nil
}

func clearTables(pool *pgxpool.Pool, tables []string) error {
	err := pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(
			context.Background(),
			fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY ",
				strings.Join(tables, ", "))); err != nil {
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
		"season_team_standings",
		"races",
		"team_drivers",
		"events",
		"season_driver_standings",
		"teams",
		"car_classes_to_car_models",
		"simulation_car_aliases",
		"seasons",
		"simulation_track_layout_aliases",
		"car_models",
		"simulation_driver_aliases",
		"series",
		"track_layouts",
		"car_brands",
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
