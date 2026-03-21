//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"

	"github.com/srlmgr/backend/authn"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
)

func countTeamRows(t *testing.T) int {
	t.Helper()

	var count int
	if err := testPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM teams").
		Scan(&count); err != nil {
		t.Fatalf("failed to count team rows: %v", err)
	}

	return count
}

func TestTeamSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (teamSetterBuilder{}).Build(&v1.CreateTeamRequest{
		SeasonId: 7,
		Name:     "Red Bull Racing",
		IsActive: true,
	})

	if !setter.SeasonID.IsValue() || setter.SeasonID.MustGet() != 7 {
		t.Fatalf("unexpected season_id setter value: %+v", setter.SeasonID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "Red Bull Racing" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.IsActive.IsValue() || !setter.IsActive.MustGet() {
		t.Fatalf("unexpected is_active setter value: %+v", setter.IsActive)
	}
}

func TestTeamSetterBuilderBuildZeroValues(t *testing.T) {
	t.Parallel()

	setter := (teamSetterBuilder{}).Build(&v1.CreateTeamRequest{
		SeasonId: 0,
		Name:     "",
		IsActive: false,
	})

	if setter.SeasonID.IsValue() {
		t.Fatalf("expected season_id to be unset, got: %+v", setter.SeasonID)
	}
	if setter.Name.IsValue() {
		t.Fatalf("expected name to be unset, got: %+v", setter.Name)
	}
	if setter.IsActive.IsValue() {
		t.Fatalf("expected is_active to be unset, got: %+v", setter.IsActive)
	}
}

func TestCreateTeamSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "rFactor 2")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateTeam(ctx, connect.NewRequest(&v1.CreateTeamRequest{
		SeasonId: uint32(season.ID),
		Name:     "Alpine Racing",
		IsActive: true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetTeam().GetName() != "Alpine Racing" {
		t.Fatalf("unexpected team name: %q", resp.Msg.GetTeam().GetName())
	}
	if resp.Msg.GetTeam().GetSeasonId() != uint32(season.ID) {
		t.Fatalf(
			"unexpected season id: got %d want %d",
			resp.Msg.GetTeam().GetSeasonId(),
			season.ID,
		)
	}

	id := int32(resp.Msg.GetTeam().GetId())
	stored, err := repo.Teams().Teams().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created team: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if stored.SeasonID != season.ID {
		t.Fatalf("unexpected stored season id: got %d want %d", stored.SeasonID, season.ID)
	}
}

func TestCreateTeamFailureDuplicateNameSameSeason(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "LMP2 Series")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	seedTeam(t, repo, season.ID, "Mercedes AMG")

	_, err := svc.CreateTeam(
		context.Background(),
		connect.NewRequest(&v1.CreateTeamRequest{
			SeasonId: uint32(season.ID),
			Name:     "Mercedes AMG",
			IsActive: true,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
	if got := countTeamRows(t); got != 1 {
		t.Fatalf("unexpected team count after duplicate create: got %d want %d", got, 1)
	}
}

func TestCreateTeamSuccessDuplicateNameDifferentSeason(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Assetto Corsa Competizione")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "GT World Challenge")
	firstSeason := seedSeason(t, repo, series.ID, ps.ID, "Season 2024")
	secondSeason := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	seedTeam(t, repo, firstSeason.ID, "Ferrari 296 GT3")

	resp, err := svc.CreateTeam(
		context.Background(),
		connect.NewRequest(&v1.CreateTeamRequest{
			SeasonId: uint32(secondSeason.ID),
			Name:     "Ferrari 296 GT3",
			IsActive: true,
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetTeam().GetSeasonId() != uint32(secondSeason.ID) {
		t.Fatalf(
			"unexpected season id: got %d want %d",
			resp.Msg.GetTeam().GetSeasonId(),
			secondSeason.ID,
		)
	}
	if got := countTeamRows(t); got != 2 {
		t.Fatalf("unexpected team count after cross-season create: got %d want %d", got, 2)
	}
}

func TestCreateTeamFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateTeam(
		context.Background(),
		connect.NewRequest(&v1.CreateTeamRequest{
			SeasonId: 1,
			Name:     "Prototype Team",
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

func TestUpdateTeamSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Le Mans Ultimate")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "Hypercar Series")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	initial := seedTeam(t, repo, season.ID, "Toyota Gazoo Racing")
	before, err := repo.Teams().Teams().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial team: %v", err)
	}

	resp, err := svc.UpdateTeam(ctx, connect.NewRequest(&v1.UpdateTeamRequest{
		TeamId:   uint32(initial.ID),
		SeasonId: uint32(season.ID),
		Name:     "Toyota Gazoo Racing Updated",
		IsActive: true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetTeam().GetName() != "Toyota Gazoo Racing Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetTeam().GetName())
	}

	after, err := repo.Teams().Teams().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated team: %v", err)
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
}

func TestUpdateTeamFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateTeam(
		context.Background(),
		connect.NewRequest(&v1.UpdateTeamRequest{
			TeamId: 999,
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

func TestUpdateTeamFailureDuplicateNameSameSeason(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Automobilista 2")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "GT4 Series")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	first := seedTeam(t, repo, season.ID, "Porsche Cayman")
	second := seedTeam(t, repo, season.ID, "BMW M4")

	_, err := svc.UpdateTeam(
		context.Background(),
		connect.NewRequest(&v1.UpdateTeamRequest{
			TeamId: uint32(second.ID),
			Name:   first.Name,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Teams().Teams().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load team after duplicate update: %v", loadErr)
	}
	if stored.Name != "BMW M4" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"BMW M4",
		)
	}
}

func TestDeleteTeamSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "rFactor 2")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "Formula Series")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	initial := seedTeam(t, repo, season.ID, "Delete Me Team")

	resp, err := svc.DeleteTeam(
		context.Background(),
		connect.NewRequest(&v1.DeleteTeamRequest{
			TeamId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Teams().Teams().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteTeamFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteTeam(
		context.Background(),
		connect.NewRequest(&v1.DeleteTeamRequest{
			TeamId: 1,
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
