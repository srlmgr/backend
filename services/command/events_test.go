//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
)

//nolint:whitespace // multiline signature with named return keeps lll and golines happy.
func seedEvent(
	t *testing.T,
	repo rootrepo.Repository,
	seasonID int32,
	trackLayoutID int32,
	name string,
) (
	event *models.Event,
) {
	t.Helper()

	var err error
	event, err = repo.Events().Create(context.Background(), &models.EventSetter{
		SeasonID:        omit.From(seasonID),
		TrackLayoutID:   omit.From(trackLayoutID),
		Name:            omit.From(name),
		EventDate:       omit.From(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:          omit.From("scheduled"),
		ProcessingState: omit.From("draft"),
		CreatedBy:       omit.From(testUserSeed),
		UpdatedBy:       omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed event %q: %v", name, err)
	}

	return event
}

func TestEventSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	eventDate := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	ts := timestamppb.New(eventDate)

	setter := (eventSetterBuilder{}).Build(&v1.CreateEventRequest{
		SeasonId:        10,
		TrackLayoutId:   20,
		Name:            "Round 1",
		EventDate:       ts,
		Status:          "planned",
		ProcessingState: "idle",
	})

	if !setter.SeasonID.IsValue() || setter.SeasonID.MustGet() != 10 {
		t.Fatalf("unexpected season_id setter value: %+v", setter.SeasonID)
	}
	if !setter.TrackLayoutID.IsValue() || setter.TrackLayoutID.MustGet() != 20 {
		t.Fatalf("unexpected track_layout_id setter value: %+v", setter.TrackLayoutID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "Round 1" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.EventDate.IsValue() {
		t.Fatal("expected event_date to be set")
	}
	if got := setter.EventDate.MustGet(); !got.Equal(eventDate) {
		t.Fatalf("unexpected event_date: got %v want %v", got, eventDate)
	}
	if !setter.Status.IsValue() || setter.Status.MustGet() != "planned" {
		t.Fatalf("unexpected status setter value: %+v", setter.Status)
	}
	if !setter.ProcessingState.IsValue() || setter.ProcessingState.MustGet() != "idle" {
		t.Fatalf("unexpected processing_state setter value: %+v", setter.ProcessingState)
	}
}

func TestEventSetterBuilderBuildNilEventDate(t *testing.T) {
	t.Parallel()

	setter := (eventSetterBuilder{}).Build(&v1.CreateEventRequest{
		SeasonId:      10,
		TrackLayoutId: 20,
		Name:          "Round 2",
		EventDate:     nil,
	})

	if setter.EventDate.IsValue() {
		t.Fatalf("expected event_date to be unset, got: %+v", setter.EventDate)
	}
}

func TestCreateEventSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	eventDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	resp, err := svc.CreateEvent(ctx, connect.NewRequest(&v1.CreateEventRequest{
		SeasonId:        uint32(season.ID),
		TrackLayoutId:   uint32(layout.ID),
		Name:            "Round 1",
		EventDate:       timestamppb.New(eventDate),
		Status:          "scheduled",
		ProcessingState: "draft",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetEvent().GetName() != "Round 1" {
		t.Fatalf("unexpected event name: %q", resp.Msg.GetEvent().GetName())
	}
	if resp.Msg.GetEvent().GetSeasonId() != uint32(season.ID) {
		t.Fatalf(
			"unexpected season_id: got %d want %d",
			resp.Msg.GetEvent().GetSeasonId(),
			season.ID,
		)
	}
	if resp.Msg.GetEvent().GetTrackLayoutId() != uint32(layout.ID) {
		t.Fatalf(
			"unexpected track_layout_id: got %d want %d",
			resp.Msg.GetEvent().GetTrackLayoutId(),
			layout.ID,
		)
	}

	id := int32(resp.Msg.GetEvent().GetId())
	stored, err := repo.Events().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created event: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if stored.SeasonID != season.ID {
		t.Fatalf("unexpected stored season_id: got %d want %d", stored.SeasonID, season.ID)
	}
	if stored.TrackLayoutID != layout.ID {
		t.Fatalf(
			"unexpected stored track_layout_id: got %d want %d",
			stored.TrackLayoutID,
			layout.ID,
		)
	}
}

func TestCreateEventFailureDuplicateNameSameSeason(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Automobilista 2")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	ps := seedPointSystem(t, repo, "Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025")
	track := seedTrack(t, repo, "Interlagos")
	layout := seedTrackLayout(t, repo, track.ID, "Full")
	seedEvent(t, repo, season.ID, layout.ID, "Round 1")

	_, err := svc.CreateEvent(
		context.Background(),
		connect.NewRequest(&v1.CreateEventRequest{
			SeasonId:        uint32(season.ID),
			TrackLayoutId:   uint32(layout.ID),
			Name:            "Round 1",
			EventDate:       timestamppb.New(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)),
			Status:          "scheduled",
			ProcessingState: "draft",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreateEventSuccessDuplicateNameDifferentSeason(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "rFactor 2")
	series := seedSeries(t, repo, sim.ID, "LMP Series")
	ps := seedPointSystem(t, repo, "Points")
	season1 := seedSeason(t, repo, series.ID, ps.ID, "2024")
	season2 := seedSeason(t, repo, series.ID, ps.ID, "2025")
	track := seedTrack(t, repo, "Le Mans")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	seedEvent(t, repo, season1.ID, layout.ID, "Round 1")

	resp, err := svc.CreateEvent(
		context.Background(),
		connect.NewRequest(&v1.CreateEventRequest{
			SeasonId:        uint32(season2.ID),
			TrackLayoutId:   uint32(layout.ID),
			Name:            "Round 1",
			EventDate:       timestamppb.New(time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)),
			Status:          "scheduled",
			ProcessingState: "draft",
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetEvent().GetSeasonId() != uint32(season2.ID) {
		t.Fatalf(
			"unexpected season_id: got %d want %d",
			resp.Msg.GetEvent().GetSeasonId(),
			season2.ID,
		)
	}
}

func TestCreateEventFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateEvent(
		context.Background(),
		connect.NewRequest(&v1.CreateEventRequest{
			SeasonId:      1,
			TrackLayoutId: 1,
			Name:          "Round 1",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeInternal)
	}
	if !errors.Is(err, txErr) {
		t.Fatalf("expected wrapped transaction error: %v", err)
	}
}

func TestUpdateEventSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "NASCAR Cup")
	ps := seedPointSystem(t, repo, "NASCAR Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Daytona")
	layout := seedTrackLayout(t, repo, track.ID, "Oval")
	initial := seedEvent(t, repo, season.ID, layout.ID, "Daytona 500")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	before, err := repo.Events().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial event: %v", err)
	}

	resp, err := svc.UpdateEvent(ctx, connect.NewRequest(&v1.UpdateEventRequest{
		EventId:         uint32(initial.ID),
		SeasonId:        uint32(season.ID),
		TrackLayoutId:   uint32(layout.ID),
		Name:            "Daytona 500 Updated",
		Status:          "completed",
		ProcessingState: "finalized",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetEvent().GetName() != "Daytona 500 Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetEvent().GetName())
	}

	after, err := repo.Events().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated event: %v", err)
	}
	if after.UpdatedBy != testUserEditor {
		t.Fatalf("unexpected UpdatedBy: got %q want %q", after.UpdatedBy, testUserEditor)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf(
			"expected UpdatedAt to move forward: before=%s after=%s",
			before.UpdatedAt,
			after.UpdatedAt,
		)
	}
	if after.Status != "completed" {
		t.Fatalf("unexpected status after update: got %q want %q", after.Status, "completed")
	}
}

func TestUpdateEventFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateEvent(
		context.Background(),
		connect.NewRequest(&v1.UpdateEventRequest{
			EventId: 999999,
			Name:    "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateEventFailureDuplicateNameSameSeason(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Assetto Corsa")
	series := seedSeries(t, repo, sim.ID, "GT Series")
	ps := seedPointSystem(t, repo, "Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025")
	track := seedTrack(t, repo, "Nurburgring")
	layout := seedTrackLayout(t, repo, track.ID, "GP")
	first := seedEvent(t, repo, season.ID, layout.ID, "Round 1")
	second := seedEvent(t, repo, season.ID, layout.ID, "Round 2")

	_, err := svc.UpdateEvent(
		context.Background(),
		connect.NewRequest(&v1.UpdateEventRequest{
			EventId: uint32(second.ID),
			Name:    first.Name,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Events().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load event after duplicate update: %v", loadErr)
	}
	if stored.Name != "Round 2" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"Round 2",
		)
	}
}

func TestDeleteEventSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Le Mans Ultimate")
	series := seedSeries(t, repo, sim.ID, "Hypercar Series")
	ps := seedPointSystem(t, repo, "Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025")
	track := seedTrack(t, repo, "Le Mans Circuit")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	initial := seedEvent(t, repo, season.ID, layout.ID, "Delete Me")

	resp, err := svc.DeleteEvent(
		context.Background(),
		connect.NewRequest(&v1.DeleteEventRequest{
			EventId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Events().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteEventFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteEvent(
		context.Background(),
		connect.NewRequest(&v1.DeleteEventRequest{
			EventId: 1,
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeInternal)
	}
	if !errors.Is(err, txErr) {
		t.Fatalf("expected wrapped transaction error: %v", err)
	}
}
