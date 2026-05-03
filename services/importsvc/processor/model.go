package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/samber/lo"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/importsvc/points"
)

type (
	EventProcInfo struct {
		Event               *models.Event
		Races               []*models.Race
		Grids               []*models.RaceGrid
		Season              *models.Season
		Series              *models.Series
		PointSystem         *models.PointSystem
		PointSystemSettings *points.PointSystemSettings
	}
	EventProcInfoCollector struct {
		repos repository.Repository
	}
)

var (
	ErrRaceNotFound = fmt.Errorf("race not found")
	ErrGridNotFound = fmt.Errorf("grid not found")
)

func NewEventProcInfoCollector(repos repository.Repository) *EventProcInfoCollector {
	return &EventProcInfoCollector{
		repos: repos,
	}
}

//nolint:whitespace,funlen //editor/linter issue
func (e *EventProcInfoCollector) ForEvent(ctx context.Context, eventID int32) (
	*EventProcInfo, error,
) {
	event, err := e.repos.Events().LoadByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	season, err := e.repos.Seasons().LoadByID(ctx, event.SeasonID)
	if err != nil {
		return nil, err
	}
	series, err := e.repos.Series().LoadByID(ctx, season.SeriesID)
	if err != nil {
		return nil, err
	}

	pointSystem, err := e.repos.PointSystems().PointSystems().LoadByID(
		ctx, season.PointSystemID)
	if err != nil {
		return nil, err
	}
	pointSystemSettings, err := e.pointSystemSettingsFromDB(pointSystem)
	if err != nil {
		return nil, err
	}

	races, err := e.repos.Races().Races().LoadByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	grids := make([]*models.RaceGrid, 0)
	for rIdx := range races {
		raceGrids, err := e.repos.Races().RaceGrids().LoadByRaceID(ctx, races[rIdx].ID)
		if err != nil {
			return nil, err
		}
		grids = append(grids, raceGrids...)
	}
	return &EventProcInfo{
		Event:               event,
		Races:               races,
		Grids:               grids,
		Season:              season,
		Series:              series,
		PointSystem:         pointSystem,
		PointSystemSettings: pointSystemSettings,
	}, nil
}

//nolint:tagliatelle // external definition
type pointRuleMetadata struct {
	RaceSettingName string          `json:"race_setting_name,omitempty"`
	Policy          json.RawMessage `json:"policy,omitempty"`
}

//nolint:whitespace,funlen // editor/linter issue
func (e *EventProcInfoCollector) pointSystemSettingsFromDB(
	ps *models.PointSystem,
) (*points.PointSystemSettings, error) {
	if ps == nil {
		return nil, nil
	}

	settings := &points.PointSystemSettings{
		Eligibility: points.EligibilitySettings{
			RaceDistPct: ps.RaceDistancePCT.InexactFloat64(),
			Guests:      ps.GuestPoints,
		},
	}

	rules := make([]*models.PointRule, 0, len(ps.R.PointRules))
	for _, rule := range ps.R.PointRules {
		if rule != nil {
			rules = append(rules, rule)
		}
	}
	if len(rules) == 0 {
		return nil, nil
	}

	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].RaceNo == rules[j].RaceNo {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].RaceNo < rules[j].RaceNo
	})

	raceIndexByNo := make(map[int32]int)
	for _, rule := range rules {
		idx, ok := raceIndexByNo[rule.RaceNo]
		if !ok {
			settings.Races = append(settings.Races, points.RaceSettings{})
			idx = len(settings.Races) - 1
			raceIndexByNo[rule.RaceNo] = idx
		}

		raceSettings := &settings.Races[idx]
		if err := e.applyPointRuleToRaceSettings(rule, raceSettings); err != nil {
			return nil, fmt.Errorf("decode point rule %d: %w", rule.ID, err)
		}
	}

	for i := range settings.Races {
		normalizeRaceSettings(&settings.Races[i])
	}

	return settings, nil
}

//nolint:whitespace,funlen // editor/linter issue
func (e *EventProcInfoCollector) applyPointRuleToRaceSettings(
	rule *models.PointRule,
	raceSettings *points.RaceSettings,
) error {
	if rule == nil || raceSettings == nil {
		return nil
	}

	metadata, err := decodePointRuleMetadata(rule.MetadataJSON.Val)
	if err != nil {
		return err
	}
	if raceSettings.Name == "" && metadata.RaceSettingName != "" {
		raceSettings.Name = metadata.RaceSettingName
	}

	policy, err := decodePointPolicySettings(rule.PointPolicy, metadata.Policy)
	if err != nil {
		return err
	}

	policyType, err := mapPolicyType(policy.GetName())
	if err != nil {
		return err
	}

	if !lo.Contains(raceSettings.Policies, policyType) {
		raceSettings.Policies = append(raceSettings.Policies, policyType)
	}

	switch cfg := policy.GetConfig().(type) {
	case *commonv1.PointPolicySettings_FinishPos:
		upsertAwardTables(raceSettings, policyType, cfg.FinishPos.GetTables())
	case *commonv1.PointPolicySettings_QualificationPos:
		upsertAwardTables(raceSettings, policyType, cfg.QualificationPos.GetTables())
	case *commonv1.PointPolicySettings_LeastIncidents:
		upsertAwardTables(raceSettings, policyType, cfg.LeastIncidents.GetTables())
	case *commonv1.PointPolicySettings_FastestLap:
		upsertAwardTables(raceSettings, policyType, cfg.FastestLap.GetTables())
	case *commonv1.PointPolicySettings_TopNFinisher:
		upsertAwardTables(raceSettings, policyType, cfg.TopNFinisher.GetTables())
	case *commonv1.PointPolicySettings_IncidentsExceeded:
		upsertThresholdPenalties(raceSettings, policyType, cfg.IncidentsExceeded.GetRules())
	default:
		// Keep policy in the policy list even if it has no config payload.
	}

	return nil
}

func decodePointRuleMetadata(raw json.RawMessage) (pointRuleMetadata, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return pointRuleMetadata{}, nil
	}

	var metadata pointRuleMetadata
	if err := json.Unmarshal(raw, &metadata); err == nil {
		if metadata.RaceSettingName != "" || len(metadata.Policy) > 0 {
			return metadata, nil
		}
	} else {
		return pointRuleMetadata{}, fmt.Errorf("unmarshal point rule metadata: %w", err)
	}

	// Backward compatibility: allow metadata_json to contain only policy JSON.
	return pointRuleMetadata{Policy: raw}, nil
}

//nolint:whitespace // editor/linter issue
func decodePointPolicySettings(
	rawPolicy string,
	encoded json.RawMessage,
) (*commonv1.PointPolicySettings, error) {
	policy := &commonv1.PointPolicySettings{}
	if len(bytes.TrimSpace(encoded)) > 0 {
		if err := protojson.Unmarshal(encoded, policy); err != nil {
			return nil, fmt.Errorf("decode point policy settings: %w", err)
		}
	}

	if enumVal, ok := commonv1.PointPolicy_value[rawPolicy]; ok {
		policy.Name = commonv1.PointPolicy(enumVal)
	}

	return policy, nil
}

func mapPolicyType(policy commonv1.PointPolicy) (points.PointPolicyType, error) {
	//nolint:exhaustive // allow unknown policy types for forward compatibility
	switch policy {
	case commonv1.PointPolicy_POINT_POLICY_FINISH_POS:
		return points.PointsPolicyFinishPos, nil
	case commonv1.PointPolicy_POINT_POLICY_FASTEST_LAP:
		return points.PointsPolicyFastestLap, nil
	case commonv1.PointPolicy_POINT_POLICY_LEAST_INCIDENTS:
		return points.PointsPolicyLeastIncidents, nil
	case commonv1.PointPolicy_POINT_POLICY_INCIDENTS_EXCEEDED:
		return points.PointsPolicyIncidentsExceeded, nil
	case commonv1.PointPolicy_POINT_POLICY_QUALIFICATION_POS:
		return points.PointsPolicyQualificationPos, nil
	case commonv1.PointPolicy_POINT_POLICY_TOP_N_FINISHER:
		return points.PointsPolicyTopNFinishers, nil
	default:
		return 0, fmt.Errorf("unsupported point policy: %s", policy.String())
	}
}

//nolint:whitespace // editor/linter issue
func upsertAwardTables(
	raceSettings *points.RaceSettings,
	policyType points.PointPolicyType,
	tables []*commonv1.PointTable,
) {
	ensureAwardSettingsLen(raceSettings, len(tables))
	for idx, table := range tables {
		vals := make([]points.PointType, 0, len(table.GetValues()))
		for _, value := range table.GetValues() {
			vals = append(vals, points.PointType(value))
		}
		raceSettings.AwardSettings[idx].Points[policyType] = vals
	}
}

//nolint:whitespace // editor/linter issue
func upsertThresholdPenalties(
	raceSettings *points.RaceSettings,
	policyType points.PointPolicyType,
	rules []*commonv1.ThresholdPenaltyRule,
) {
	ensurePenaltySettingsLen(raceSettings, len(rules))
	for idx, rule := range rules {
		//nolint:lll // long line
		raceSettings.PenaltySettings[idx].Arguments[policyType] = points.ThresholdPenaltySettings{
			Threshold:  int32(rule.GetThreshold()),
			PenaltyPct: rule.GetPenaltyPercent(),
		}
	}
}

func ensureAwardSettingsLen(raceSettings *points.RaceSettings, minLen int) {
	for len(raceSettings.AwardSettings) < minLen {
		raceSettings.AwardSettings = append(
			raceSettings.AwardSettings,
			points.RankedPolicySettings{Points: map[points.PointPolicyType][]points.PointType{}},
		)
	}
}

func ensurePenaltySettingsLen(raceSettings *points.RaceSettings, minLen int) {
	for len(raceSettings.PenaltySettings) < minLen {
		raceSettings.PenaltySettings = append(
			raceSettings.PenaltySettings,
			points.PointPenaltySettings{Arguments: map[points.PointPolicyType]any{}},
		)
	}
}

func normalizeRaceSettings(raceSettings *points.RaceSettings) {
	gridCount := len(raceSettings.AwardSettings)
	if len(raceSettings.PenaltySettings) > gridCount {
		gridCount = len(raceSettings.PenaltySettings)
	}
	if gridCount == 0 {
		gridCount = 1
	}

	ensureAwardSettingsLen(raceSettings, gridCount)
	ensurePenaltySettingsLen(raceSettings, gridCount)
}

func (epi *EventProcInfo) ResolverFunc(ctx context.Context) points.ResolveGridID {
	return func(gridID int32) (raceNo, gridNo int32, err error) {
		grid, ok := lo.Find(epi.Grids, func(item *models.RaceGrid) bool {
			return item.ID == gridID
		})
		if !ok {
			return 0, 0, ErrGridNotFound
		}
		race, ok := lo.Find(epi.Races, func(item *models.Race) bool {
			return item.ID == grid.RaceID
		})
		if !ok {
			return 0, 0, ErrRaceNotFound
		}
		return race.SequenceNo - 1, grid.SequenceNo - 1, nil
	}
}

//nolint:whitespace,unused //editor/linter issue, to be deleted later
func (e *EventProcInfoCollector) fakePointSystemSettings(
	ps *models.PointSystem,
	season *models.Season,
) *points.PointSystemSettings {
	switch ps.Name {
	case "Standard", "VRPC":
		return e.StandardPointSystemSettings(season)
	case "VRGES":
		return e.VRGESPointSystemSettings()
	default:
		return e.StandardPointSystemSettings(season)
	}
}

//nolint:whitespace,funlen // tmp method will be replaced by database values later
func (e *EventProcInfoCollector) StandardPointSystemSettings(
	season *models.Season,
) *points.PointSystemSettings {
	settings := &points.PointSystemSettings{
		Eligibility: points.EligibilitySettings{
			RaceDistPct: 0.75,
		},
		Races: []points.RaceSettings{
			{
				Policies: []points.PointPolicyType{
					points.PointsPolicyFinishPos,
					points.PointsPolicyQualificationPos,
					points.PointsPolicyLeastIncidents,
					points.PointsPolicyFastestLap,
				},
				AwardSettings: []points.RankedPolicySettings{
					{
						Points: map[points.PointPolicyType][]points.PointType{
							points.PointsPolicyFinishPos: {
								100,
								95,
								92,
								90,
								88,
								86,
								84,
								82,
								80,
								78,
								76,
								74,
								72,
								70,
								68,
								66,
								64,
								62,
								60,
								58,
								56,
								54,
								52,
								50,
								48,
								46,
								44,
								42,
								40,
							},
							points.PointsPolicyQualificationPos: {3, 2, 1},
							points.PointsPolicyLeastIncidents:   {3, 2, 1},
							points.PointsPolicyFastestLap:       {1},
						},
					},
				},
				PenaltySettings: []points.PointPenaltySettings{
					{Arguments: map[points.PointPolicyType]any{}},
				},
			},
			{
				Policies: []points.PointPolicyType{
					points.PointsPolicyFinishPos,
					points.PointsPolicyLeastIncidents,
					points.PointsPolicyFastestLap,
				},
				AwardSettings: []points.RankedPolicySettings{
					{
						Points: map[points.PointPolicyType][]points.PointType{
							points.PointsPolicyFinishPos: {
								100,
								95,
								92,
								90,
								88,
								86,
								84,
								82,
								80,
								78,
								76,
								74,
								72,
								70,
								68,
								66,
								64,
								62,
								60,
								58,
								56,
								54,
								52,
								50,
								48,
								46,
								44,
								42,
								40,
							},

							points.PointsPolicyLeastIncidents: {3, 2, 1},
							points.PointsPolicyFastestLap:     {1},
						},
					},
				},
				PenaltySettings: []points.PointPenaltySettings{
					{Arguments: map[points.PointPolicyType]any{}},
				},
			},
		},
	}
	return settings
}

//nolint:lll,funlen // tmp method will be replaced by database values later
func (e *EventProcInfoCollector) VRGESPointSystemSettings() *points.PointSystemSettings {
	settings := &points.PointSystemSettings{
		Eligibility: points.EligibilitySettings{
			RaceDistPct: 0.75,
		},
		Races: []points.RaceSettings{
			{
				Policies: []points.PointPolicyType{
					points.PointsPolicyFinishPos,
					points.PointsPolicyLeastIncidents,
					points.PointsPolicyIncidentsExceeded,
				},
				AwardSettings: []points.RankedPolicySettings{
					{
						Points: map[points.PointPolicyType][]points.PointType{
							points.PointsPolicyFinishPos: {
								50,
								45,
								41,
								38,
								36,
								34,
								32,
								30,
								28,
								26,
								25,
								24,
								23,
								22,
								21,
								20,
								19,
								18,
								17,
								16,
								15,
								14,
								13,
								12,
								11,
								10,
								9,
								8,
								7,
								6,
								5,
								4,
								3,
								2,
								1,
							},
							points.PointsPolicyLeastIncidents: {3, 2, 1},
						},
					},
				},
				PenaltySettings: []points.PointPenaltySettings{
					{
						Arguments: map[points.PointPolicyType]any{
							points.PointsPolicyIncidentsExceeded: points.ThresholdPenaltySettings{
								Threshold:  30,
								PenaltyPct: 0.1,
							},
						},
					},
				},
			},
		},
	}
	return settings
}
