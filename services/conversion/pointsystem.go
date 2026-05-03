package conversion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	bobtypes "github.com/stephenafamo/bob/types"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
)

// pointRuleMetadata stores race-setting metadata and policy configuration.
//
// This bridges proto message structure (PointRaceSettings with nested Policies)
// and database schema (PointRule rows with metadata_json field).
//
// The metadata_json column in point_rules stores a JSON representation that
// enables nested proto messages to be decomposed and reconstructed:
//
//   - RaceSettingName: User-facing name for the race setting
//     (e.g., "Sprint Points - Race 1"). Not stored in schema as a dedicated column;
//     must be persisted through this metadata JSON field.
//
//   - Policy: Encoded PointPolicySettings protobuf, serialized as JSON bytes.
//     Contains policy type (e.g., POINT_POLICY_FINISH_POS) and configuration
//     (e.g., point tables, bonus rules). Policy name stored separately in
//     point_policy column for easy querying.
//
// Example JSON representation in metadata_json:
//
//	{
//	  "race_setting_name": "Sprint Points - Race 1",
//	  "policy": {
//	    "name": "POINT_POLICY_FINISH_POS",
//	    "config": {"finishPos": {"tables": [{"values": [100, 95, 92]}]}}
//	  }
//	}
//
// Usage Flow:
//
//  1. Command: User sends CreatePointSystemRequest with PointRaceSettings array
//     → MarshalPointRuleMetadata encodes each race setting into metadata_json
//     → Each policy creates one point_rules row with race_no and metadata
//
//  2. Query: User requests GetPointSystem by ID
//     → Repository loads point_system and preloads child point_rules
//     → PointSystemToPointSystem calls pointSystemRaceSettings
//     → For each point_rules row, decode JSON back to proto
//     → Race settings reconstructed by grouping rules by race_no
//
//nolint:tagliatelle // internal struct for metadata encoding/decoding
type pointRuleMetadata struct {
	RaceSettingName string          `json:"race_setting_name,omitempty"`
	Policy          json.RawMessage `json:"policy,omitempty"`
}

func (s *Service) pointPolicyFromString(value string) commonv1.PointPolicy {
	policy, ok := commonv1.PointPolicy_value[value]
	if !ok {
		return commonv1.PointPolicy_POINT_POLICY_UNSPECIFIED
	}

	return commonv1.PointPolicy(policy)
}

// MarshalPointRuleMetadata encodes a race setting name and policy to
// database-persistent JSON.
//
// Called during Create/UpdatePointSystem when decomposing nested
// PointRaceSettings into individual point_rules rows. Each policy within a
// race setting becomes a separate point_rules row with metadata encoding the
// race-setting name (for display) and full PointPolicySettings protobuf.
//
// Result is stored in point_rules.metadata_json and later decoded by
// UnmarshalPointRuleMetadata when reconstructing race settings in responses.
//
// Args:
//
//	raceSettingName: Display name of the race setting (user-provided context)
//	policy: Full PointPolicySettings including type and config details
//
// Returns:
//
//	Encoded JSON wrapped in bob.types.JSON for database insertion
//	Error if proto marshaling or JSON encoding fails
//
//nolint:whitespace // editor/linter issue
func (s *Service) MarshalPointRuleMetadata(
	raceSettingName string,
	policy *commonv1.PointPolicySettings,
) (bobtypes.JSON[json.RawMessage], error) {
	metadata := pointRuleMetadata{RaceSettingName: raceSettingName}
	if policy != nil {
		encodedPolicy, err := protojson.Marshal(policy)
		if err != nil {
			return bobtypes.JSON[json.RawMessage]{}, fmt.Errorf(
				"marshal point policy settings: %w", err,
			)
		}
		metadata.Policy = json.RawMessage(encodedPolicy)
	}

	encodedMetadata, err := json.Marshal(metadata)
	if err != nil {
		return bobtypes.JSON[json.RawMessage]{}, fmt.Errorf(
			"marshal point rule metadata: %w", err,
		)
	}

	return bobtypes.JSON[json.RawMessage]{
		Val: json.RawMessage(encodedMetadata),
	}, nil
}

// decodePointRuleMetadata deserializes JSON-encoded metadata back into
// structured form.
//
// Internal helper called during query/conversion phase when reconstructing
// PointRaceSettings from stored point_rules rows. Decodes metadata_json field
// and extracts race-setting name and policy configuration.
//
// If metadata is empty or malformed, gracefully returns empty metadata struct
// or treats raw bytes as a policy field, avoiding query failures on corrupted
// data.
//
// Args:
//
//	raw: JSON bytes from point_rules.metadata_json
//
// Returns:
//
//	Decoded pointRuleMetadata with RaceSettingName and Policy fields
//	Error only on critical JSON parsing failures (warnings in caller)
//
//nolint:unparam // error always nil for now, kept for future use
func decodePointRuleMetadata(raw json.RawMessage) (pointRuleMetadata, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return pointRuleMetadata{}, nil
	}

	var metadata pointRuleMetadata
	if err := json.Unmarshal(raw, &metadata); err == nil {
		if metadata.RaceSettingName != "" || len(metadata.Policy) > 0 {
			return metadata, nil
		}
	}

	return pointRuleMetadata{Policy: raw}, nil
}

// pointPolicySettingsFromRule reconstructs a PointPolicySettings proto from
// a stored PointRule.
//
// Counterpart to MarshalPointRuleMetadata: decodes metadata_json and extracts
// both the race-setting name (for PointRaceSettings.Name) and full policy
// configuration proto (for PointRaceSettings.Policies).
//
// Policy type (Name field) from rule.PointPolicy (enum name as string for DB
// querying); detailed configuration from decoded metadata JSON. This separation
// enables both efficient filtering and full data retention.
//
// Args:
//
//	rule: Database model with metadata_json and point_policy columns
//
// Returns:
//
//	raceSettingName: Display name of race setting (empty if not set)
//	policy: Reconstructed PointPolicySettings with Name and Config
//	error: Metadata decoding or proto unmarshaling failures (non-critical;
//	       caller logs warning)
//
//nolint:whitespace // editor/linter issue
func (s *Service) pointPolicySettingsFromRule(
	rule *models.PointRule,
) (string, *commonv1.PointPolicySettings, error) {
	if rule == nil {
		return "", nil, nil
	}

	metadata, err := decodePointRuleMetadata(rule.MetadataJSON.Val)
	if err != nil {
		return "", nil, fmt.Errorf("decode point rule metadata: %w", err)
	}

	policy := &commonv1.PointPolicySettings{}
	if len(bytes.TrimSpace(metadata.Policy)) > 0 {
		if err := protojson.Unmarshal(metadata.Policy, policy); err != nil {
			return "", nil, fmt.Errorf(
				"decode point policy settings: %w", err,
			)
		}
	}
	policy.Name = s.pointPolicyFromString(rule.PointPolicy)

	return metadata.RaceSettingName, policy, nil
}

// pointSystemRaceSettings reconstructs the nested PointRaceSettings array
// from database rows.
//
// Main orchestrator function called during PointSystemToPointSystem conversion
// (used by GetPointSystem and ListPointSystems responses). Takes a PointSystem
// model with preloaded point_rules children and rebuilds the original nested
// message structure by:
//
//  1. Sorting rules by race_no (implicit ordering from Create/Update)
//  2. Grouping rules by race_no (same race_no = policies in same race setting)
//  3. For each rule, unmarshaling metadata to recover race-setting name
//  4. Assembling groups into PointRaceSettings with nested Policies arrays
//
// Restores structure decomposed during command persistence, ensuring queries
// return the same nested shape as the client sent.
//
// Example:
//
//	Input DB: [PointRule{race_no=0, name="Sprint 1", ...},
//	           PointRule{race_no=0, name="Sprint 1", ...}]
//	Output: [PointRaceSettings{name="Sprint 1", policies=[Policy1, Policy2]}]
//
// Args:
//
//	model: PointSystem with preloaded R.PointRules from repository
//
// Returns:
//
//	Reconstructed race settings; nil if no rules; logs warnings on decode errors
//
//nolint:funlen,whitespace // orchestration requires multiple steps
func (s *Service) pointSystemRaceSettings(
	model *models.PointSystem,
) []*commonv1.PointRaceSettings {
	if model == nil || len(model.R.PointRules) == 0 {
		return nil
	}

	rules := make([]*models.PointRule, 0, len(model.R.PointRules))
	for _, rule := range model.R.PointRules {
		if rule != nil {
			rules = append(rules, rule)
		}
	}
	if len(rules) == 0 {
		return nil
	}

	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].RaceNo == rules[j].RaceNo {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].RaceNo < rules[j].RaceNo
	})

	raceSettings := make([]*commonv1.PointRaceSettings, 0)
	var currentRaceNo int32
	var current *commonv1.PointRaceSettings

	for index, rule := range rules {
		if index == 0 || rule.RaceNo != currentRaceNo {
			currentRaceNo = rule.RaceNo
			current = &commonv1.PointRaceSettings{}
			raceSettings = append(raceSettings, current)
		}

		raceSettingName, policy, err := s.pointPolicySettingsFromRule(rule)
		if err != nil {
			s.logger.Warn(
				"failed to decode point rule metadata",
				log.ErrorField(err),
			)
			policyName := s.pointPolicyFromString(rule.PointPolicy)
			policy = &commonv1.PointPolicySettings{Name: policyName}
		}

		if current.Name == "" && raceSettingName != "" {
			current.Name = raceSettingName
		}
		if policy != nil {
			current.Policies = append(current.Policies, policy)
		}
	}

	return raceSettings
}
