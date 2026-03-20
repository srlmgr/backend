package testdb

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	tcpg "github.com/srlmgr/backend/testsupport/tcpostgres"
)

func InitTestDB() (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var setupErr error
	if os.Getenv("TESTDB_URL") != "" {
		pool, setupErr = tcpg.SetupExternalTestDB()
	} else {
		pool, setupErr = tcpg.SetupTestDB()
	}
	if setupErr != nil {
		return nil, setupErr
	}
	if err := pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
		return tcpg.ClearAllTables(pool)
	}); err != nil {
		log.Fatalf("initTestDb: %v\n", err)
	}

	return pool, nil
}

func ClearAllTables(pool *pgxpool.Pool) error {
	return tcpg.ClearAllTables(pool)
}
