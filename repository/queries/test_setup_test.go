package queries

import (
	"os"
	"testing"

	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/repository/testhelpers"
	"github.com/srlmgr/backend/testsupport/testdb"
)

func TestMain(m *testing.M) {
	pool, err := testdb.InitTestDB()
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	testhelpers.TestPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func newDBBackedRepository(t *testing.T) rootrepo.Queries {
	t.Helper()
	testhelpers.ResetTestTables(t)
	t.Cleanup(func() {
		testhelpers.ResetTestTables(t)
	})

	return New(testhelpers.TestPool)
}
