package events

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/db/models"
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

func newDBBackedRepository(t *testing.T) Repository {
	t.Helper()
	testhelpers.ResetTestTables(t)
	t.Cleanup(func() {
		testhelpers.ResetTestTables(t)
	})

	return New(testhelpers.TestPool)
}

//nolint:whitespace // multiline signature style
func seedEvent(
	t *testing.T,
	repo Repository,
	seasonID int32,
	trackLayoutID int32,
	name string,
) (
	event *models.Event,
) {
	t.Helper()

	var err error
	event, err = repo.Create(context.Background(), &models.EventSetter{
		SeasonID:      omit.From(seasonID),
		TrackLayoutID: omit.From(trackLayoutID),
		Name:          omit.From(name),
		EventDate:     omit.From(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		CreatedBy:     omit.From(testhelpers.TestUserSeed),
		UpdatedBy:     omit.From(testhelpers.TestUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed event %q: %v", name, err)
	}

	return event
}
