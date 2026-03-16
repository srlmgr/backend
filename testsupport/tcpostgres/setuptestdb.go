//nolint:errcheck // testsetup
package tcpostgres

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/srlmgr/backend/db/migrate"
	database "github.com/srlmgr/backend/db/postgres"
)

// create a pg connection pool for the iracelog testdatabase
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
		WithName("iracelog-service-manager-test"),
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

func ClearEventTable(pool *pgxpool.Pool) {
	if _, err := pool.Exec(context.Background(),
		"delete from event"); err != nil {
		log.Fatalf("ClearEventTable: %v\n", err)
	}
}

func ClearCarTable(pool *pgxpool.Pool) {
	if _, err := pool.Exec(context.Background(),
		"delete from car_state_proto"); err != nil {
		log.Fatalf("ClearCarStateProtoTable: %v\n", err)
	}
}

func ClearTrackTable(pool *pgxpool.Pool) {
	pool.Exec(context.Background(), "delete from track")
}

func ClearDriverTable(pool *pgxpool.Pool) {
	pool.Exec(context.Background(), "delete from driver")
}

func ClearSpeedmapTable(pool *pgxpool.Pool) {
	if _, err := pool.Exec(context.Background(),
		"delete from speedmap_proto"); err != nil {
		log.Fatalf("ClearSpeedmapProtoTable: %v\n", err)
	}
}

func ClearAnalysisTable(pool *pgxpool.Pool) {
	pool.Exec(context.Background(), "delete from analysis")
}

func ClearStateDataTable(pool *pgxpool.Pool) {
	if _, err := pool.Exec(context.Background(),
		"delete from race_state_proto"); err != nil {
		log.Fatalf("ClearStateDataTable: %v\n", err)
	}
}

func clearTables(pool *pgxpool.Pool, tables []string) error {
	err := pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
		for _, table := range tables {
			if _, err := tx.Exec(context.Background(), "delete from "+table); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func ClearAllTables(pool *pgxpool.Pool) {
	tables := []string{
		"drivers",
		"point_systems",
		"racing_sims",
	}
	err := clearTables(pool, tables)
	if err != nil {
		log.Fatalf("ClearAllTables: %v\n", err)
	}
}
