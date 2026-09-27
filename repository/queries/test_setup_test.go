package queries

import (
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/testhelpers"
	"github.com/srlmgr/backend/testsupport/testdb"
)

var packagePool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, cleanup, err := testdb.InitTestDB()
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	packagePool = pool
	code := m.Run()
	pool.Close()
	cleanup()
	os.Exit(code)
}

func newDBBackedRepository(t *testing.T) rootrepo.Queries {
	t.Helper()

	pool := testhelpers.NewTestPool(t, packagePool)

	return New(pool)
}
