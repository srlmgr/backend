//nolint:lll,dupl,funlen // test files
package command

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
	"github.com/srlmgr/backend/services/conversion"
)

//nolint:whitespace // editor/linter issue
func seedRace(
	t *testing.T,
	repo rootrepo.Repository,
	eventID int32,
	name, sessionType string,
	sequenceNo int32,
) *models.Race {
	t.Helper()
	race, err := repo.Races().Races().Create(context.Background(), &models.RaceSetter{
		EventID:     omit.From(eventID),
		Name:        omit.From(name),
		SessionType: omit.From(sessionType),
		SequenceNo:  omit.From(sequenceNo),
		CreatedBy:   omit.From(testUserSeed),
		UpdatedBy:   omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed race %q: %v", name, err)
	}
	return race
}

func TestRaceSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter, err := (raceSetterBuilder{}).Build(&v1.CreateRaceRequest{
		EventId:     5,
		Name:        "Qualifying 1",
		SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING,
		SequenceNo:  1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !setter.EventID.IsValue() || setter.EventID.MustGet() != 5 {
		t.Fatalf("unexpected event_id setter value: %+v", setter.EventID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "Qualifying 1" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.SessionType.IsValue() ||
		setter.SessionType.MustGet() != conversion.RaceSessionTypeQualifying {

		t.Fatalf("unexpected session_type setter value: %+v", setter.SessionType)
	}
	if !setter.SequenceNo.IsValue() || setter.SequenceNo.MustGet() != 1 {
		t.Fatalf("unexpected sequence_no setter value: %+v", setter.SequenceNo)
	}
}

func TestRaceSetterBuilderBuildZeroValues(t *testing.T) {
	t.Parallel()

	setter, err := (raceSetterBuilder{}).Build(&v1.CreateRaceRequest{
		EventId:     0,
		Name:        "",
		SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_UNSPECIFIED,
		SequenceNo:  0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if setter.EventID.IsValue() {
		t.Fatalf("expected event_id to be unset, got: %+v", setter.EventID)
	}
	if setter.Name.IsValue() {
		t.Fatalf("expected name to be unset, got: %+v", setter.Name)
	}
	if setter.SessionType.IsValue() {
		t.Fatalf("expected session_type to be unset, got: %+v", setter.SessionType)
	}
	if setter.SequenceNo.IsValue() {
		t.Fatalf("expected sequence_no to be unset, got: %+v", setter.SequenceNo)
	}
}

func TestRaceSetterBuilderBuildInvalidSessionType(t *testing.T) {
	t.Parallel()

	_, err := (raceSetterBuilder{}).Build(&v1.CreateRaceRequest{
		SessionType: commonv1.RaceSessionType(99),
	})
	if err == nil {
		t.Fatal("expected error for invalid session type")
	}
}

func TestCreateRaceSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateRace(ctx, connect.NewRequest(&v1.CreateRaceRequest{
		EventId:     uint32(event.ID),
		Name:        "Qualifying 1",
		SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING,
		SequenceNo:  1,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetRace().GetName() != "Qualifying 1" {
		t.Fatalf("unexpected race name: %q", resp.Msg.GetRace().GetName())
	}
	if resp.Msg.GetRace().GetEventId() != uint32(event.ID) {
		t.Fatalf("unexpected event_id: got %d want %d", resp.Msg.GetRace().GetEventId(), event.ID)
	}
	if resp.Msg.GetRace().
		GetSessionType() !=
		commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING {

		t.Fatalf("unexpected session_type: got %v", resp.Msg.GetRace().GetSessionType())
	}
	if resp.Msg.GetRace().GetSequenceNo() != 1 {
		t.Fatalf("unexpected sequence_no: got %d want 1", resp.Msg.GetRace().GetSequenceNo())
	}

	id := int32(resp.Msg.GetRace().GetId())
	stored, err := repo.Races().Races().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created race: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if stored.EventID != event.ID {
		t.Fatalf("unexpected stored event_id: got %d want %d", stored.EventID, event.ID)
	}
}

func TestCreateRaceFailureDuplicateNameSameEvent(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeQualifying, 1)

	_, err := svc.CreateRace(
		context.Background(),
		connect.NewRequest(&v1.CreateRaceRequest{
			EventId:     uint32(event.ID),
			Name:        "Race 1",
			SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING,
			SequenceNo:  2,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreateRaceFailureDuplicateSequenceSameEvent(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeQualifying, 1)

	_, err := svc.CreateRace(
		context.Background(),
		connect.NewRequest(&v1.CreateRaceRequest{
			EventId:     uint32(event.ID),
			Name:        "Race 2",
			SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING,
			SequenceNo:  1,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate sequence error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreateRaceSuccessDuplicateNameDifferentEvent(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event1 := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	event2 := seedEvent(t, repo, season.ID, layout.ID, "Round 2", 2)
	seedRace(t, repo, event1.ID, "Race 1", conversion.RaceSessionTypeQualifying, 1)

	resp, err := svc.CreateRace(
		context.Background(),
		connect.NewRequest(&v1.CreateRaceRequest{
			EventId:     uint32(event2.ID),
			Name:        "Race 1",
			SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING,
			SequenceNo:  1,
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetRace().GetEventId() != uint32(event2.ID) {
		t.Fatalf("unexpected event_id: got %d want %d", resp.Msg.GetRace().GetEventId(), event2.ID)
	}
}

func TestCreateRaceFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateRace(
		context.Background(),
		connect.NewRequest(&v1.CreateRaceRequest{
			EventId:     1,
			Name:        "Race 1",
			SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING,
			SequenceNo:  1,
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

func TestUpdateRaceSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	initial := seedRace(t, repo, event.ID, "Qualifying 1", conversion.RaceSessionTypeQualifying, 1)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	before, err := repo.Races().Races().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial race: %v", err)
	}

	// Sleep briefly so UpdatedAt can advance
	time.Sleep(10 * time.Millisecond)

	resp, err := svc.UpdateRace(ctx, connect.NewRequest(&v1.UpdateRaceRequest{
		RaceId:      uint32(initial.ID),
		Name:        "Heat 1 Updated",
		SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE,
		SequenceNo:  2,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetRace().GetName() != "Heat 1 Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetRace().GetName())
	}
	if resp.Msg.GetRace().GetSessionType() != commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE {
		t.Fatalf("unexpected session_type: got %v", resp.Msg.GetRace().GetSessionType())
	}
	if resp.Msg.GetRace().GetSequenceNo() != 2 {
		t.Fatalf("unexpected sequence_no: got %d want 2", resp.Msg.GetRace().GetSequenceNo())
	}

	after, err := repo.Races().Races().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated race: %v", err)
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
	if after.Name != "Heat 1 Updated" {
		t.Fatalf("unexpected name after update: got %q want %q", after.Name, "Heat 1 Updated")
	}
	if after.SessionType != conversion.RaceSessionTypeRace {
		t.Fatalf(
			"unexpected session_type after update: got %q want %q",
			after.SessionType,
			conversion.RaceSessionTypeRace,
		)
	}
	if after.SequenceNo != 2 {
		t.Fatalf("unexpected sequence_no after update: got %d want 2", after.SequenceNo)
	}
}

func TestUpdateRaceFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateRace(
		context.Background(),
		connect.NewRequest(&v1.UpdateRaceRequest{
			RaceId: 999999,
			Name:   "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateRaceFailureDuplicateNameSameEvent(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	first := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeQualifying, 1)
	second := seedRace(t, repo, event.ID, "Race 2", conversion.RaceSessionTypeQualifying, 2)

	_, err := svc.UpdateRace(
		context.Background(),
		connect.NewRequest(&v1.UpdateRaceRequest{
			RaceId: uint32(second.ID),
			Name:   first.Name,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Races().Races().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load race after duplicate update: %v", loadErr)
	}
	if stored.Name != "Race 2" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"Race 2",
		)
	}
}

func TestUpdateRaceFailureDuplicateSequenceSameEvent(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	first := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeQualifying, 1)
	second := seedRace(t, repo, event.ID, "Race 2", conversion.RaceSessionTypeQualifying, 2)

	_, err := svc.UpdateRace(
		context.Background(),
		connect.NewRequest(&v1.UpdateRaceRequest{
			RaceId:     uint32(second.ID),
			SequenceNo: first.SequenceNo,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate sequence error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestDeleteRaceSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	initial := seedRace(t, repo, event.ID, "Delete Me", conversion.RaceSessionTypeQualifying, 1)

	resp, err := svc.DeleteRace(
		context.Background(),
		connect.NewRequest(&v1.DeleteRaceRequest{
			RaceId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Races().Races().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteRaceFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteRace(
		context.Background(),
		connect.NewRequest(&v1.DeleteRaceRequest{
			RaceId: 1,
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

func TestRaceGridSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter, err := (raceGridSetterBuilder{}).Build(&v1.CreateRaceGridRequest{
		RaceId:      5,
		Name:        "Grid 1",
		SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE,
		SequenceNo:  1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if setter.RaceID.IsValue() {
		t.Fatalf("expected race_id to be unset in builder, got: %+v", setter.RaceID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "Grid 1" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.SessionType.IsValue() ||
		setter.SessionType.MustGet() != conversion.RaceSessionTypeRace {

		t.Fatalf("unexpected session_type setter value: %+v", setter.SessionType)
	}
	if !setter.SequenceNo.IsValue() || setter.SequenceNo.MustGet() != 1 {
		t.Fatalf("unexpected sequence_no setter value: %+v", setter.SequenceNo)
	}
}

func TestRaceGridSetterBuilderBuildInvalidSessionType(t *testing.T) {
	t.Parallel()

	_, err := (raceGridSetterBuilder{}).Build(&v1.CreateRaceGridRequest{
		SessionType: commonv1.RaceSessionType(99),
	})
	if err == nil {
		t.Fatal("expected error for invalid session type")
	}
}

func TestCreateRaceGridSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	race := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeRace, 1)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateRaceGrid(ctx, connect.NewRequest(&v1.CreateRaceGridRequest{
		RaceId:      uint32(race.ID),
		Name:        "Grid 1",
		SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE,
		SequenceNo:  1,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetRaceGrid().GetName() != "Grid 1" {
		t.Fatalf("unexpected race grid name: %q", resp.Msg.GetRaceGrid().GetName())
	}
	if resp.Msg.GetRaceGrid().GetRaceId() != uint32(race.ID) {
		t.Fatalf("unexpected race_id: got %d want %d", resp.Msg.GetRaceGrid().GetRaceId(), race.ID)
	}
	if resp.Msg.GetRaceGrid().GetSessionType() != commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE {
		t.Fatalf("unexpected session_type: got %v", resp.Msg.GetRaceGrid().GetSessionType())
	}

	id := int32(resp.Msg.GetRaceGrid().GetId())
	stored, err := repo.Races().RaceGrids().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created race grid: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if stored.RaceID != race.ID {
		t.Fatalf("unexpected stored race_id: got %d want %d", stored.RaceID, race.ID)
	}
}

func TestUpdateRaceGridSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	race := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeRace, 1)
	initial := seedRaceGrid(t, repo, race.ID, "Grid 1", conversion.RaceSessionTypeRace, 1)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	before, err := repo.Races().RaceGrids().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial race grid: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	resp, err := svc.UpdateRaceGrid(ctx, connect.NewRequest(&v1.UpdateRaceGridRequest{
		RaceGridId:  uint32(initial.ID),
		Name:        "Grid 1 Updated",
		SessionType: commonv1.RaceSessionType_RACE_SESSION_TYPE_HEAT,
		SequenceNo:  2,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetRaceGrid().GetName() != "Grid 1 Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetRaceGrid().GetName())
	}
	if resp.Msg.GetRaceGrid().GetSessionType() != commonv1.RaceSessionType_RACE_SESSION_TYPE_HEAT {
		t.Fatalf("unexpected session_type: got %v", resp.Msg.GetRaceGrid().GetSessionType())
	}
	if resp.Msg.GetRaceGrid().GetSequenceNo() != 2 {
		t.Fatalf("unexpected sequence_no: got %d want 2", resp.Msg.GetRaceGrid().GetSequenceNo())
	}

	after, err := repo.Races().RaceGrids().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated race grid: %v", err)
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
	if after.Name != "Grid 1 Updated" {
		t.Fatalf("unexpected name after update: got %q want %q", after.Name, "Grid 1 Updated")
	}
	if after.SessionType != conversion.RaceSessionTypeHeat {
		t.Fatalf(
			"unexpected session_type after update: got %q want %q",
			after.SessionType,
			conversion.RaceSessionTypeHeat,
		)
	}
	if after.SequenceNo != 2 {
		t.Fatalf("unexpected sequence_no after update: got %d want 2", after.SequenceNo)
	}
}

func TestUpdateRaceGridFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateRaceGrid(
		context.Background(),
		connect.NewRequest(&v1.UpdateRaceGridRequest{
			RaceGridId: 999999,
			Name:       "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestDeleteRaceGridSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	series := seedSeries(t, repo, sim.ID, "Porsche Cup")
	ps := seedPointSystem(t, repo, "Standard Points")
	season := seedSeason(t, repo, series.ID, ps.ID, "2025 Season")
	track := seedTrack(t, repo, "Spa-Francorchamps")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")
	event := seedEvent(t, repo, season.ID, layout.ID, "Round 1", 1)
	race := seedRace(t, repo, event.ID, "Race 1", conversion.RaceSessionTypeRace, 1)
	initial := seedRaceGrid(t, repo, race.ID, "Delete Me", conversion.RaceSessionTypeRace, 1)

	resp, err := svc.DeleteRaceGrid(
		context.Background(),
		connect.NewRequest(&v1.DeleteRaceGridRequest{
			RaceGridId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Races().RaceGrids().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteRaceGridFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteRaceGrid(
		context.Background(),
		connect.NewRequest(&v1.DeleteRaceGridRequest{
			RaceGridId: 1,
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
