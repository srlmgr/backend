package testdb

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	tcpg "github.com/srlmgr/backend/testsupport/tcpostgres"
)

// InitTestDB provisions a database for a test package, to be called once from
// TestMain. When TESTDB_URL is set it uses that external, shared database as-is (the
// returned cleanup is a no-op). Otherwise it clones a fresh database from the shared
// template on the testcontainers-managed postgres instance; the returned cleanup
// drops that clone.
func InitTestDB() (*pgxpool.Pool, func(), error) {
	if os.Getenv("TESTDB_URL") != "" {
		pool, err := tcpg.SetupExternalTestDB()
		if err != nil {
			return nil, nil, err
		}
		if err := tcpg.ClearAllTables(pool); err != nil {
			return nil, nil, err
		}
		return pool, func() {}, nil
	}

	ctx := context.Background()

	adminURL, err := tcpg.SetupTestDB()
	if err != nil {
		return nil, nil, err
	}

	dbName, dbURL, err := tcpg.CloneDatabase(ctx, adminURL)
	if err != nil {
		return nil, nil, err
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		pool.Close()
		_ = tcpg.DropDatabase(context.Background(), adminURL, dbName)
	}

	return pool, cleanup, nil
}

// NewTestDatabase returns an isolated pool for a single test, to be called from
// within a test body (needs *testing.T for cleanup registration). When
// testcontainers manages postgres, it clones a fresh database from the template and
// drops it via t.Cleanup. When TESTDB_URL is set (external DB, no clone support), it
// falls back to truncating all tables on packagePool (the pool returned by
// InitTestDB) and returns that same pool.
func NewTestDatabase(t *testing.T, packagePool *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()

	if os.Getenv("TESTDB_URL") != "" {
		if err := tcpg.ClearAllTables(packagePool); err != nil {
			t.Fatalf("reset shared test database: %v", err)
		}
		t.Cleanup(func() {
			_ = tcpg.ClearAllTables(packagePool)
		})
		return packagePool
	}

	ctx := context.Background()

	adminURL, err := tcpg.SetupTestDB()
	if err != nil {
		t.Fatalf("setup test database: %v", err)
	}

	dbName, dbURL, err := tcpg.CloneDatabase(ctx, adminURL)
	if err != nil {
		t.Fatalf("clone test database: %v", err)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to cloned test database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping cloned test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_ = tcpg.DropDatabase(context.Background(), adminURL, dbName)
	})

	return pool
}

func ClearAllTables(pool *pgxpool.Pool) error {
	return tcpg.ClearAllTables(pool)
}
