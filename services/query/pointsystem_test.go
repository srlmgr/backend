//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package query

import (
	"context"
	"errors"
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"github.com/shopspring/decimal"

	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/conversion"
)

func seedPointSystem(t *testing.T, repo rootrepo.Repository, name string) *models.PointSystem {
	t.Helper()
	ps, err := repo.PointSystems().
		PointSystems().
		Create(context.Background(), &models.PointSystemSetter{
			Name:      omit.From(name),
			CreatedBy: omit.From(testUserSeed),
			UpdatedBy: omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed point system %q: %v", name, err)
	}
	return ps
}

func TestListPointSystemsEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListPointSystems(
		context.Background(),
		connect.NewRequest(&queryv1.ListPointSystemsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListPointSystemsReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	alpha := seedPointSystem(t, repo, "Alpha Points")
	beta := seedPointSystem(t, repo, "Beta Points")

	resp, err := svc.ListPointSystems(
		context.Background(),
		connect.NewRequest(&queryv1.ListPointSystemsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	ids := make(map[uint32]bool)
	for _, item := range items {
		ids[item.GetId()] = true
	}

	if !ids[uint32(alpha.ID)] {
		t.Errorf("alpha point system (id=%d) not found in response", alpha.ID)
	}
	if !ids[uint32(beta.ID)] {
		t.Errorf("beta point system (id=%d) not found in response", beta.ID)
	}
}

//nolint:funlen,govet // much work to do
func TestGetPointSystemSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	ps, err := repo.PointSystems().
		PointSystems().
		Create(context.Background(), &models.PointSystemSetter{
			Name:            omit.From("Sprint Points"),
			GuestPoints:     omit.From(true),
			RaceDistancePCT: omit.From(decimal.NewFromFloat(0.75)),
			CreatedBy:       omit.From(testUserSeed),
			UpdatedBy:       omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed point system: %v", err)
	}
	codec := conversion.New()
	metadata, err := codec.MarshalPointRuleMetadata(
		"Settings for race 1",
		&commonv1.PointPolicySettings{
			Name: commonv1.PointPolicy_POINT_POLICY_FINISH_POS,
			Config: &commonv1.PointPolicySettings_FinishPos{
				FinishPos: &commonv1.PositionPointsConfig{
					Tables: []*commonv1.PointTable{{Values: []int32{100, 95, 92}}},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to seed point rule metadata: %v", err)
	}
	if _, err := repo.PointSystems().
		PointRules().
		Create(context.Background(), &models.PointRuleSetter{
			PointSystemID: omit.From(ps.ID),
			RaceNo:        omit.From(int32(0)),
			PointPolicy:   omit.From(commonv1.PointPolicy_POINT_POLICY_FINISH_POS.String()),
			MetadataJSON:  omit.From(metadata),
			CreatedBy:     omit.From(testUserSeed),
			UpdatedBy:     omit.From(testUserSeed),
		}); err != nil {
		t.Fatalf("failed to seed point rule: %v", err)
	}

	resp, err := svc.GetPointSystem(
		context.Background(),
		connect.NewRequest(&queryv1.GetPointSystemRequest{
			Id: uint32(ps.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.GetPointSystem().GetId() != uint32(ps.ID) {
		t.Errorf("expected id %d, got %d", ps.ID, resp.Msg.GetPointSystem().GetId())
	}
	if resp.Msg.GetPointSystem().GetName() != "Sprint Points" {
		t.Errorf("expected name %q, got %q", "Sprint Points", resp.Msg.GetPointSystem().GetName())
	}
	if !resp.Msg.GetPointSystem().GetEligibility().GetGuests() {
		t.Fatal("expected guests eligibility to be returned")
	}
	if len(resp.Msg.GetPointSystem().GetRaceSettings()) != 1 {
		t.Fatalf(
			"unexpected race settings count: %d",
			len(resp.Msg.GetPointSystem().GetRaceSettings()),
		)
	}
	if resp.Msg.GetPointSystem().GetRaceSettings()[0].GetPolicies()[0].GetName() != commonv1.PointPolicy_POINT_POLICY_FINISH_POS {
		t.Fatalf(
			"unexpected first policy: %v",
			resp.Msg.GetPointSystem().GetRaceSettings()[0].GetPolicies()[0].GetName(),
		)
	}
}

func TestGetPointSystemNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetPointSystem(
		context.Background(),
		connect.NewRequest(&queryv1.GetPointSystemRequest{
			Id: 99999,
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
