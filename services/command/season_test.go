//nolint:lll,dupl,funlen,gocyclo // test files can have some duplication and long lines for test data setup
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

func TestSeasonSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (seasonSetterBuilder{}).Build(&v1.CreateSeasonRequest{
		Name:           "2024 Season",
		PointSystemId:  3,
		HasTeams:       true,
		IsTeamBased:    false,
		IsMulticlass:   false,
		TeamPointsTopN: 2,
		SkipEvents:     2,
		Status:         "active",
	})

	if !setter.Name.IsValue() || setter.Name.MustGet() != "2024 Season" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.PointSystemID.IsValue() || setter.PointSystemID.MustGet() != 3 {
		t.Fatalf("unexpected point_system_id setter value: %+v", setter.PointSystemID)
	}
	if !setter.HasTeams.IsValue() || !setter.HasTeams.MustGet() {
		t.Fatalf("unexpected has_teams setter value: %+v", setter.HasTeams)
	}
	if !setter.IsTeamBased.IsValue() || setter.IsTeamBased.MustGet() {
		t.Fatalf("unexpected is_team_based setter value: %+v", setter.IsTeamBased)
	}
	if !setter.IsMulticlass.IsValue() || setter.IsMulticlass.MustGet() {
		t.Fatalf("unexpected is_multiclass setter value: %+v", setter.IsMulticlass)
	}

	if !setter.TeamPointsTopN.IsValue() || setter.TeamPointsTopN.MustGet() != 2 {
		t.Fatalf("unexpected team_points_top_n setter value: %+v", setter.TeamPointsTopN)
	}

	if !setter.SkipEvents.IsValue() || setter.SkipEvents.MustGet() != 2 {
		t.Fatalf("unexpected skip_events setter value: %+v", setter.SkipEvents)
	}
	if !setter.Status.IsValue() || setter.Status.MustGet() != "active" {
		t.Fatalf("unexpected status setter value: %+v", setter.Status)
	}
}

func TestSeasonSetterBuilderBuildZeroValues(t *testing.T) {
	t.Parallel()

	setter := (seasonSetterBuilder{}).Build(&v1.CreateSeasonRequest{})

	if setter.SeriesID.IsValue() {
		t.Fatalf("expected series_id to be unset, got %+v", setter.SeriesID)
	}
	if setter.Name.IsValue() {
		t.Fatalf("expected name to be unset, got %+v", setter.Name)
	}
	if setter.PointSystemID.IsValue() {
		t.Fatalf("expected point_system_id to be unset, got %+v", setter.PointSystemID)
	}
	if !setter.HasTeams.IsValue() || setter.HasTeams.MustGet() {
		t.Fatalf("expected has_teams to be set to false, got %+v", setter.HasTeams)
	}
	if setter.SkipEvents.IsValue() {
		t.Fatalf("expected skip_events to be unset, got %+v", setter.SkipEvents)
	}
	if setter.Status.IsValue() {
		t.Fatalf("expected status to be unset, got %+v", setter.Status)
	}
}

func TestCreateSeasonSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "rFactor 2")
	series := seedSeries(t, repo, sim.ID, "GT3")
	ps := seedPointSystem(t, repo, "Formula Points")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateSeason(ctx, connect.NewRequest(&v1.CreateSeasonRequest{
		SeriesId:      uint32(series.ID),
		Name:          "2024 GT3 Season",
		PointSystemId: uint32(ps.ID),
		HasTeams:      true,
		Status:        "active",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetSeason().GetName() != "2024 GT3 Season" {
		t.Fatalf("unexpected season name: %q", resp.Msg.GetSeason().GetName())
	}
	if resp.Msg.GetSeason().GetSeriesId() != uint32(series.ID) {
		t.Fatalf(
			"unexpected series id: got %d want %d",
			resp.Msg.GetSeason().GetSeriesId(),
			series.ID,
		)
	}
	if resp.Msg.GetSeason().GetPointSystemId() != uint32(ps.ID) {
		t.Fatalf(
			"unexpected point_system id: got %d want %d",
			resp.Msg.GetSeason().GetPointSystemId(),
			ps.ID,
		)
	}

	id := int32(resp.Msg.GetSeason().GetId())
	stored, err := repo.Seasons().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created season: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if stored.SeriesID != series.ID {
		t.Fatalf("unexpected stored series id: got %d want %d", stored.SeriesID, series.ID)
	}
	if stored.PointSystemID != ps.ID {
		t.Fatalf(
			"unexpected stored point_system id: got %d want %d",
			stored.PointSystemID,
			ps.ID,
		)
	}
}

func TestCreateSeasonFailureDuplicateNameSameSeries(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Assetto Corsa Competizione")
	series := seedSeries(t, repo, sim.ID, "GT3")
	ps := seedPointSystem(t, repo, "Formula Points")
	seedSeason(t, repo, series.ID, ps.ID, "Season 1")

	_, err := svc.CreateSeason(
		context.Background(),
		connect.NewRequest(&v1.CreateSeasonRequest{
			SeriesId:      uint32(series.ID),
			Name:          "Season 1",
			PointSystemId: uint32(ps.ID),
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreateSeasonSuccessDuplicateNameDifferentSeries(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	firstSeries := seedSeries(t, repo, sim.ID, "GT3")
	secondSeries := seedSeries(t, repo, sim.ID, "GT4")
	ps := seedPointSystem(t, repo, "Formula Points")
	seedSeason(t, repo, firstSeries.ID, ps.ID, "Season 1")

	resp, err := svc.CreateSeason(
		context.Background(),
		connect.NewRequest(&v1.CreateSeasonRequest{
			SeriesId:      uint32(secondSeries.ID),
			Name:          "Season 1",
			PointSystemId: uint32(ps.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetSeason().GetSeriesId() != uint32(secondSeries.ID) {
		t.Fatalf(
			"unexpected series id: got %d want %d",
			resp.Msg.GetSeason().GetSeriesId(),
			secondSeries.ID,
		)
	}
}

func TestCreateSeasonFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateSeason(
		context.Background(),
		connect.NewRequest(&v1.CreateSeasonRequest{
			SeriesId:      1,
			Name:          "Season X",
			PointSystemId: 1,
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

func TestUpdateSeasonSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Le Mans Ultimate")
	series := seedSeries(t, repo, sim.ID, "LMP2")
	ps := seedPointSystem(t, repo, "F1 Points")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	initial := seedSeason(t, repo, series.ID, ps.ID, "2023 LMP2")
	before, err := repo.Seasons().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial season: %v", err)
	}

	resp, err := svc.UpdateSeason(ctx, connect.NewRequest(&v1.UpdateSeasonRequest{
		SeasonId:      uint32(initial.ID),
		Name:          "2023 LMP2 Updated",
		PointSystemId: uint32(ps.ID),
		Status:        "completed",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetSeason().GetName() != "2023 LMP2 Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetSeason().GetName())
	}

	after, err := repo.Seasons().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated season: %v", err)
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
		t.Fatalf("unexpected status after update: %q", after.Status)
	}
}

func TestUpdateSeasonFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateSeason(
		context.Background(),
		connect.NewRequest(&v1.UpdateSeasonRequest{
			SeasonId: 999,
			Name:     "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateSeasonFailureDuplicateNameSameSeries(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Automobilista 2")
	series := seedSeries(t, repo, sim.ID, "GT3")
	ps := seedPointSystem(t, repo, "Formula Points")
	first := seedSeason(t, repo, series.ID, ps.ID, "Season 1")
	second := seedSeason(t, repo, series.ID, ps.ID, "Season 2")

	_, err := svc.UpdateSeason(
		context.Background(),
		connect.NewRequest(&v1.UpdateSeasonRequest{
			SeasonId: uint32(second.ID),
			Name:     first.Name,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Seasons().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load season after duplicate update: %v", loadErr)
	}
	if stored.Name != "Season 2" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"Season 2",
		)
	}
}

func TestDeleteSeasonSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "rFactor 2")
	series := seedSeries(t, repo, sim.ID, "GT3")
	ps := seedPointSystem(t, repo, "Formula Points")
	initial := seedSeason(t, repo, series.ID, ps.ID, "Delete Me")

	resp, err := svc.DeleteSeason(
		context.Background(),
		connect.NewRequest(&v1.DeleteSeasonRequest{
			SeasonId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Seasons().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteSeasonFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteSeason(
		context.Background(),
		connect.NewRequest(&v1.DeleteSeasonRequest{
			SeasonId: 1,
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
