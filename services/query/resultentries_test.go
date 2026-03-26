package query

import (
	"context"
	"errors"
	"testing"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"

	"github.com/srlmgr/backend/db/models"
	mytypes "github.com/srlmgr/backend/db/mytypes"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/conversion"
)

//nolint:whitespace // multiline signature style
func seedImportBatch(
	t *testing.T,
	repo rootrepo.Repository,
	raceID int32,
) *models.ImportBatch {
	t.Helper()

	batch, err := repo.ImportBatches().Create(
		context.Background(), &models.ImportBatchSetter{
			RaceID:          omit.From(raceID),
			ImportFormat:    omit.From(mytypes.ImportFormat(conversion.ImportFormatCSV)),
			Payload:         omit.From([]byte("{}")),
			ProcessingState: omit.From(conversion.EventProcessingStateRawImported),
			CreatedBy:       omit.From(testUserSeed),
			UpdatedBy:       omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed import batch: %v", err)
	}

	return batch
}

//nolint:whitespace // multiline signature style
func seedResultEntry(
	t *testing.T,
	repo rootrepo.Repository,
	raceID int32,
	driverName string,
	finishingPosition int32,
) *models.ResultEntry {
	t.Helper()

	entry, err := repo.ResultEntries().Create(
		context.Background(), &models.ResultEntrySetter{
			RaceID:            omit.From(raceID),
			RawDriverName:     omitnull.From(driverName),
			FinishingPosition: omit.From(finishingPosition),
			CompletedLaps:     omit.From(int32(0)),
			State:             omit.From(conversion.ResultStateNormal),
			CreatedBy:         omit.From(testUserSeed),
			UpdatedBy:         omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed result entry for %q: %v", driverName, err)
	}

	return entry
}

func TestGetResultEntrySuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	pointSystem := seedPointSystem(t, repo, "GT3 Points")
	season := seedSeason(t, repo, series.ID, pointSystem.ID, "2024 Season")
	track := seedTrack(t, repo, "Daytona")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1")
	race := seedRace(t, repo, event.ID, "Feature Race", "race", 1)
	batch := seedImportBatch(t, repo, race.ID)
	entry := seedResultEntry(t, repo, race.ID, "Alice", 1)
	_ = batch // batch is not directly relevant to this test, but seeded for completeness
	resp, err := svc.GetResultEntry(
		context.Background(),
		connect.NewRequest(&queryv1.GetResultEntryRequest{
			ResultEntryId: uint32(entry.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := resp.Msg.GetResultEntry()
	if got.GetId() != uint32(entry.ID) {
		t.Errorf("expected id %d, got %d", entry.ID, got.GetId())
	}
	if got.GetRaceId() != uint32(race.ID) {
		t.Errorf("expected race_id %d, got %d", race.ID, got.GetRaceId())
	}
	if got.GetFinishingPosition() != 1 {
		t.Errorf("expected finishing_position 1, got %d", got.GetFinishingPosition())
	}
}

func TestGetResultEntryNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetResultEntry(
		context.Background(),
		connect.NewRequest(&queryv1.GetResultEntryRequest{
			ResultEntryId: 99999,
		}),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}
