//nolint:whitespace,lll,funlen,gocyclo // tests
package processor

import (
	"encoding/json"
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/shopspring/decimal"
	bobtypes "github.com/stephenafamo/bob/types"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/services/importsvc/points"
)

func TestPointSystemSettingsFromDB_MapsPoliciesAndGridSettings(t *testing.T) {
	t.Parallel()

	collector := &EventProcInfoCollector{}

	finishPolicy := &commonv1.PointPolicySettings{
		Name: commonv1.PointPolicy_POINT_POLICY_FINISH_POS,
		Config: &commonv1.PointPolicySettings_FinishPos{
			FinishPos: &commonv1.PositionPointsConfig{
				Tables: []*commonv1.PointTable{
					{Values: []int32{10, 8, 6}},
					{Values: []int32{5, 4, 3}},
				},
			},
		},
	}
	incidentPolicy := &commonv1.PointPolicySettings{
		Name: commonv1.PointPolicy_POINT_POLICY_INCIDENTS_EXCEEDED,
		Config: &commonv1.PointPolicySettings_IncidentsExceeded{
			IncidentsExceeded: &commonv1.ThresholdPenaltyConfig{
				Rules: []*commonv1.ThresholdPenaltyRule{
					{Threshold: 3, PenaltyPercent: 0.1},
					{Threshold: 1, PenaltyPercent: 0.5},
				},
			},
		},
	}
	fastestLapPolicy := &commonv1.PointPolicySettings{
		Name: commonv1.PointPolicy_POINT_POLICY_FASTEST_LAP,
		Config: &commonv1.PointPolicySettings_FastestLap{
			FastestLap: &commonv1.PositionPointsConfig{
				Tables: []*commonv1.PointTable{{Values: []int32{1}}},
			},
		},
	}

	ps := &models.PointSystem{GuestPoints: true, RaceDistancePCT: decimal.NewFromFloat(0.75)}
	ps.R.PointRules = models.PointRuleSlice{
		newRule(t, 1, 0, "Race 1", finishPolicy),
		newRule(t, 2, 0, "Race 1", incidentPolicy),
		newRule(t, 3, 1, "Race 2", fastestLapPolicy),
	}

	settings, err := collector.pointSystemSettingsFromDB(ps)
	if err != nil {
		t.Fatalf("pointSystemSettingsFromDB returned error: %v", err)
	}
	if settings == nil {
		t.Fatal("expected non-nil settings")
	}

	if !settings.Eligibility.Guests || settings.Eligibility.RaceDistPct != 0.75 {
		t.Fatalf("unexpected eligibility: %+v", settings.Eligibility)
	}
	if len(settings.Races) != 2 {
		t.Fatalf("unexpected race count: got %d want 2", len(settings.Races))
	}

	race1 := settings.Races[0]
	if race1.Name != "Race 1" {
		t.Fatalf("unexpected race 1 name: %q", race1.Name)
	}
	if len(race1.AwardSettings) != 2 || len(race1.PenaltySettings) != 2 {
		t.Fatalf(
			"unexpected race 1 grid settings: awards=%d penalties=%d",
			len(race1.AwardSettings),
			len(race1.PenaltySettings),
		)
	}
	if got := race1.AwardSettings[0].Points[points.PointsPolicyFinishPos]; len(got) != 3 ||
		got[0] != 10 {
		t.Fatalf("unexpected race 1 grid 0 finish points: %+v", got)
	}
	pen, ok := race1.PenaltySettings[1].Arguments[points.PointsPolicyIncidentsExceeded].(points.ThresholdPenaltySettings)
	if !ok {
		t.Fatal("expected incidents exceeded penalty settings in race 1 grid 1")
	}
	if pen.Threshold != 1 || pen.PenaltyPct != 0.5 {
		t.Fatalf("unexpected incidents penalty config: %+v", pen)
	}

	race2 := settings.Races[1]
	if race2.Name != "Race 2" {
		t.Fatalf("unexpected race 2 name: %q", race2.Name)
	}
	if len(race2.AwardSettings) != 1 || len(race2.PenaltySettings) != 1 {
		t.Fatalf(
			"unexpected race 2 grid settings: awards=%d penalties=%d",
			len(race2.AwardSettings),
			len(race2.PenaltySettings),
		)
	}
	if got := race2.AwardSettings[0].Points[points.PointsPolicyFastestLap]; len(got) != 1 ||
		got[0] != 1 {
		t.Fatalf("unexpected race 2 fastest lap points: %+v", got)
	}
}

func newRule(
	t *testing.T,
	id int32,
	raceNo int32,
	raceName string,
	policy *commonv1.PointPolicySettings,
) *models.PointRule {
	t.Helper()

	policyJSON, err := protojson.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}

	metadataJSON, err := json.Marshal(pointRuleMetadata{
		RaceSettingName: raceName,
		Policy:          policyJSON,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	return &models.PointRule{
		ID:          id,
		RaceNo:      raceNo,
		PointPolicy: policy.GetName().String(),
		MetadataJSON: bobtypes.JSON[json.RawMessage]{
			Val: metadataJSON,
		},
	}
}
