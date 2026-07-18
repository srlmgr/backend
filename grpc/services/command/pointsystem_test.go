//nolint:lll,dupl,funlen // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/db/models"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
)

func TestPointSystemSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (pointSystemSetterBuilder{}).Build(&v1.CreatePointSystemRequest{
		Name:        "Formula Points",
		Description: "Standard formula points system",
		Eligibility: &commonv1.PointEligibility{
			Guests:                 true,
			MinRaceDistancePercent: 0.75,
		},
	})

	if !setter.Name.IsValue() || setter.Name.MustGet() != "Formula Points" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.Description.IsValue() ||
		setter.Description.MustGet() != "Standard formula points system" {

		t.Fatalf("unexpected description setter value: %+v", setter.Description)
	}
	if !setter.GuestPoints.IsValue() || !setter.GuestPoints.MustGet() {
		t.Fatalf("unexpected guest_points setter value: %+v", setter.GuestPoints)
	}
	if !setter.RaceDistancePCT.IsValue() ||
		setter.RaceDistancePCT.MustGet().InexactFloat64() != 0.75 {

		t.Fatalf("unexpected race_distance_pct setter value: %+v", setter.RaceDistancePCT)
	}
}

func TestPointSystemSetterBuilderBuildZeroValues(t *testing.T) {
	t.Parallel()

	setter := (pointSystemSetterBuilder{}).Build(&v1.CreatePointSystemRequest{})

	if setter.Name.IsValue() {
		t.Fatalf("expected name to be unset, got: %+v", setter.Name)
	}
	if setter.Description.IsValue() {
		t.Fatalf("expected description to be unset, got: %+v", setter.Description)
	}
}

func TestCreatePointSystemSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreatePointSystem(ctx, connect.NewRequest(&v1.CreatePointSystemRequest{
		Name:        "Sprint Points",
		Description: "Points for sprint races",
		Eligibility: &commonv1.PointEligibility{
			Guests:                 true,
			MinRaceDistancePercent: 0.75,
		},
		RaceSettings: []*commonv1.PointRaceSettings{
			{
				Name: "Settings for race 1",
				Policies: []*commonv1.PointPolicySettings{
					{
						Name: commonv1.PointPolicy_POINT_POLICY_FINISH_POS,
						Config: &commonv1.PointPolicySettings_FinishPos{
							FinishPos: &commonv1.PositionPointsConfig{
								Tables: []*commonv1.PointTable{{Values: []int32{100, 95, 92}}},
							},
						},
					},
					{
						Name: commonv1.PointPolicy_POINT_POLICY_FASTEST_LAP,
						Config: &commonv1.PointPolicySettings_FastestLap{
							FastestLap: &commonv1.PositionPointsConfig{
								Tables: []*commonv1.PointTable{{Values: []int32{1}}},
							},
						},
					},
				},
			},
			{
				Name: "Settings for race 2",
				Policies: []*commonv1.PointPolicySettings{
					{
						Name: commonv1.PointPolicy_POINT_POLICY_LEAST_INCIDENTS,
						Config: &commonv1.PointPolicySettings_LeastIncidents{
							LeastIncidents: &commonv1.PositionPointsConfig{
								Tables: []*commonv1.PointTable{{Values: []int32{3, 2, 1}}},
							},
						},
					},
				},
			},
		},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetPointSystem().GetName() != "Sprint Points" {
		t.Fatalf("unexpected point system name: %q", resp.Msg.GetPointSystem().GetName())
	}
	if resp.Msg.GetPointSystem().GetDescription() != "Points for sprint races" {
		t.Fatalf(
			"unexpected point system description: %q",
			resp.Msg.GetPointSystem().GetDescription(),
		)
	}
	if !resp.Msg.GetPointSystem().GetEligibility().GetGuests() {
		t.Fatal("expected guests eligibility to round-trip")
	}
	if len(resp.Msg.GetPointSystem().GetRaceSettings()) != 2 {
		t.Fatalf(
			"unexpected race settings count: %d",
			len(resp.Msg.GetPointSystem().GetRaceSettings()),
		)
	}
	if resp.Msg.GetPointSystem().GetRaceSettings()[0].GetName() != "Settings for race 1" {
		t.Fatalf(
			"unexpected first race setting name: %q",
			resp.Msg.GetPointSystem().GetRaceSettings()[0].GetName(),
		)
	}

	id := int32(resp.Msg.GetPointSystem().GetId())
	stored, err := repo.PointSystems().PointSystems().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created point system: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if !stored.GuestPoints || stored.RaceDistancePCT.InexactFloat64() != 0.75 {
		t.Fatalf(
			"unexpected stored eligibility: guests=%t pct=%v",
			stored.GuestPoints,
			stored.RaceDistancePCT,
		)
	}
	if len(stored.R.PointRules) != 3 {
		t.Fatalf("unexpected stored point rule count: %d", len(stored.R.PointRules))
	}
}

func TestCreatePointSystemFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedPointSystem(t, repo, "duplicate-ps")

	_, err := svc.CreatePointSystem(
		context.Background(),
		connect.NewRequest(&v1.CreatePointSystemRequest{
			Name: "duplicate-ps",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreatePointSystemFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreatePointSystem(
		context.Background(),
		connect.NewRequest(&v1.CreatePointSystemRequest{
			Name: "tx-fail-ps",
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

//nolint:govet // ok here
func TestUpdatePointSystemSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	initial := seedPointSystem(t, repo, "Original Points")
	metadata, err := svc.conversion.MarshalPointRuleMetadata(
		"Old settings",
		&commonv1.PointPolicySettings{
			Name: commonv1.PointPolicy_POINT_POLICY_FASTEST_LAP,
			Config: &commonv1.PointPolicySettings_FastestLap{
				FastestLap: &commonv1.PositionPointsConfig{
					Tables: []*commonv1.PointTable{{Values: []int32{1}}},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to build point rule metadata: %v", err)
	}
	if _, err := repo.PointSystems().
		PointRules().
		Create(context.Background(), &models.PointRuleSetter{
			PointSystemID: omit.From(initial.ID),
			RaceNo:        omit.From(int32(0)),
			PointPolicy:   omit.From(commonv1.PointPolicy_POINT_POLICY_FASTEST_LAP.String()),
			MetadataJSON:  omit.From(metadata),
			CreatedBy:     omit.From(testUserSeed),
			UpdatedBy:     omit.From(testUserSeed),
		}); err != nil {
		t.Fatalf("failed to seed initial point rule: %v", err)
	}
	before, err := repo.PointSystems().PointSystems().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial point system: %v", err)
	}

	resp, err := svc.UpdatePointSystem(ctx, connect.NewRequest(&v1.UpdatePointSystemRequest{
		PointSystemId: uint32(initial.ID),
		Name:          "Updated Points",
		Description:   "Updated description",
		Eligibility: &commonv1.PointEligibility{
			Guests:                 false,
			MinRaceDistancePercent: 0.5,
		},
		RaceSettings: []*commonv1.PointRaceSettings{
			{
				Name: "Updated race 1",
				Policies: []*commonv1.PointPolicySettings{
					{
						Name: commonv1.PointPolicy_POINT_POLICY_FINISH_POS,
						Config: &commonv1.PointPolicySettings_FinishPos{
							FinishPos: &commonv1.PositionPointsConfig{
								Tables: []*commonv1.PointTable{{Values: []int32{25, 18, 15}}},
							},
						},
					},
				},
			},
		},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetPointSystem().GetName() != "Updated Points" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetPointSystem().GetName())
	}
	if len(resp.Msg.GetPointSystem().GetRaceSettings()) != 1 {
		t.Fatalf(
			"unexpected updated race settings count: %d",
			len(resp.Msg.GetPointSystem().GetRaceSettings()),
		)
	}

	after, err := repo.PointSystems().PointSystems().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated point system: %v", err)
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
	if after.GuestPoints || after.RaceDistancePCT.InexactFloat64() != 0.5 {
		t.Fatalf(
			"unexpected updated eligibility: guests=%t pct=%v",
			after.GuestPoints,
			after.RaceDistancePCT,
		)
	}
	if len(after.R.PointRules) != 1 {
		t.Fatalf("expected point rules to be replaced, got %d rules", len(after.R.PointRules))
	}
	if after.R.PointRules[0].PointPolicy != commonv1.PointPolicy_POINT_POLICY_FINISH_POS.String() {
		t.Fatalf("unexpected remaining point policy: %q", after.R.PointRules[0].PointPolicy)
	}
}

func TestUpdatePointSystemFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdatePointSystem(
		context.Background(),
		connect.NewRequest(&v1.UpdatePointSystemRequest{
			PointSystemId: 999,
			Name:          "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdatePointSystemFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	first := seedPointSystem(t, repo, "first-ps")
	second := seedPointSystem(t, repo, "second-ps")

	_, err := svc.UpdatePointSystem(
		context.Background(),
		connect.NewRequest(&v1.UpdatePointSystemRequest{
			PointSystemId: uint32(second.ID),
			Name:          first.Name,
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.PointSystems().PointSystems().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load point system after duplicate update: %v", loadErr)
	}
	if stored.Name != "second-ps" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"second-ps",
		)
	}
}

func TestDeletePointSystemSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	initial := seedPointSystem(t, repo, "delete-me-ps")

	resp, err := svc.DeletePointSystem(
		context.Background(),
		connect.NewRequest(&v1.DeletePointSystemRequest{
			PointSystemId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.PointSystems().PointSystems().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeletePointSystemFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeletePointSystem(
		context.Background(),
		connect.NewRequest(&v1.DeletePointSystemRequest{
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
