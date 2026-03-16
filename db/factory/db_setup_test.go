// testdb_setup.go: Setup testDB for tests using TEST_DB env var
// This file initializes the testDB variable for use in tests.
// It should be imported in test files that require a database connection.

package factory

import (
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob"

	"github.com/srlmgr/backend/testsupport/testdb"
)

func init() {
	pool, err := testdb.InitTestDB()
	if err != nil {
		panic("Failed to connect to test database: " + err.Error())
	}
	testDB = bob.NewDB(stdlib.OpenDBFromPool(pool))
}
