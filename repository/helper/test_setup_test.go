package helper

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob"

	"github.com/srlmgr/backend/repository/pgbob"
	"github.com/srlmgr/backend/repository/testhelpers"
	"github.com/srlmgr/backend/testsupport/testdb"
)

var testDB bob.Transactor[bob.Tx]

func TestMain(m *testing.M) {
	pool, err := testdb.InitTestDB()
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}

	testhelpers.TestPool = pool
	testDB = bob.NewDB(stdlib.OpenDBFromPool(pool))

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func newTxContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		if err := tx.Rollback(context.Background()); err != nil {
			t.Fatalf("failed to rollback transaction: %v", err)
		}
	})

	return pgbob.NewContext(ctx, tx)
}
