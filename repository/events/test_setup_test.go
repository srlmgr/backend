package events

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/srlmgr/backend/db/models"
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

func newDBBackedRepository(t *testing.T) Repository {
	t.Helper()

	pool := testhelpers.NewTestPool(t, packagePool)

	return New(pool)
}

//nolint:whitespace // multiline signature style
func seedEvent(
	t *testing.T,
	repo Repository,
	seasonID int32,
	pointSystemID int32,
	trackLayoutID int32,
	name string,
	sequenceNo int32,
) (
	event *models.Event,
) {
	t.Helper()

	var err error
	event, err = repo.Create(context.Background(), &models.EventSetter{
		SeasonID:      omit.From(seasonID),
		PointSystemID: omit.From(pointSystemID),
		TrackLayoutID: omit.From(trackLayoutID),
		Name:          omit.From(name),
		SequenceNo:    omit.From(sequenceNo),
		EventDate:     omit.From(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		CreatedBy:     omit.From(testhelpers.TestUserSeed),
		UpdatedBy:     omit.From(testhelpers.TestUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed event %q: %v", name, err)
	}

	return event
}
