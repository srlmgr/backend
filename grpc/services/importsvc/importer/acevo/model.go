//nolint:tagliatelle,lll // external definition
package acevo

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

// GUID represents the two-part uint64 identifier used throughout the ACEvo format.
// Both parts are encoded as decimal strings in JSON.
// A holds the high 64 bits and B holds the low 64 bits of a UUID.
type GUID struct {
	A string `json:"a"`
	B string `json:"b"`
}

// UUID reconstructs the standard UUID string (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
// from the two uint64 halves. Returns an empty string on parse error.
func (g GUID) UUID() string {
	hi, err := strconv.ParseUint(g.A, 10, 64)
	if err != nil {
		return ""
	}
	lo, err := strconv.ParseUint(g.B, 10, 64)
	if err != nil {
		return ""
	}
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[0:8], hi)
	binary.BigEndian.PutUint64(buf[8:16], lo)
	return fmt.Sprintf("%08X-%04X-%04X-%04X-%012X",
		binary.BigEndian.Uint32(buf[0:4]),
		binary.BigEndian.Uint16(buf[4:6]),
		binary.BigEndian.Uint16(buf[6:8]),
		binary.BigEndian.Uint16(buf[8:10]),
		buf[10:16],
	)
}

// Session is the top-level structure of an ACEvo session result file.
type Session struct {
	ServerName      string         `json:"server_name"`
	ServerID        string         `json:"server_id"`
	ServerIP        string         `json:"server_ip"`
	SeasonGUID      string         `json:"season_guid"`
	SessionName     string         `json:"session_name"`
	SessionType     string         `json:"session_type"`
	TrackName       string         `json:"track_name"`
	TrackLayoutName string         `json:"track_layout_name"`
	ChampionshipID  GUID           `json:"championship_id"`
	EventIndex      int            `json:"event_index"`
	SessionIndex    int            `json:"session_index"`
	IsStatic        bool           `json:"is_static"`
	IsCompleted     bool           `json:"is_completed"`
	Specialization  Specialization `json:"specialization"`
	CarStandings    []CarStanding  `json:"car_standings"`
	DriverStandings []GUID         `json:"driver_standings"`
	TimeStandings   []int          `json:"time_standings"`
	Drivers         []Driver       `json:"drivers"`
	Laps            []Lap          `json:"laps"`
	CarPoints       []any          `json:"car_points"`
	DriverPoints    []any          `json:"driver_points"`
	Cars            []Car          `json:"cars"`
}

// Specialization holds session-type-specific configuration (e.g. TimeAttack or InstantRace).
// The @type field carries the protobuf type URL that identifies which session kind this is.
type Specialization struct {
	Type                         string                 `json:"@type"`
	Base                         SpecializationBase     `json:"base"`
	BaseFile                     string                 `json:"base_file"`
	BaseSource                   string                 `json:"base_source"`
	PenaltyTransformations       PenaltyTransformations `json:"penalty_transformations"`
	PenaltyTransformationsFile   string                 `json:"penalty_transformations_file"`
	PenaltyTransformationsSource string                 `json:"penalty_transformations_source"`
	PenaltyInvestigations        PenaltyInvestigations  `json:"penalty_investigations"`
	PenaltyInvestigationsFile    string                 `json:"penalty_investigations_file"`
	PenaltyInvestigationsSource  string                 `json:"penalty_investigations_source"`
	Rules                        []any                  `json:"rules"`
	AllowedCarTableKeys          []any                  `json:"allowed_car_table_keys,omitempty"`
	AllowedCarKeys               []any                  `json:"allowed_car_keys,omitempty"`
}

// SpecializationBase contains the common session parameters.
// Fields only present in race sessions (InstantRace) are omitted when empty.
type SpecializationBase struct {
	// Qualify / TimeAttack fields
	SessionDurationMs                       int    `json:"session_duration_ms"`
	SessionLaps                             int    `json:"session_laps"`
	MaximumSessionOvertimeDurationMs        int    `json:"maximum_session_overtime_duration_ms"`
	MaximumSessionOvertimeBeforeNextSession int    `json:"maximum_session_overtime_before_next_session"`
	IntroMusic                              bool   `json:"intro_music"`
	EndMusic                                bool   `json:"end_music"`
	EndReplayType                           string `json:"end_replay_type"`

	// Race / InstantRace additional fields
	CountdownTimeMs                          int    `json:"countdown_time_ms,omitempty"`
	CountdownRandomMinTimeMs                 int    `json:"countdown_random_min_time_ms,omitempty"`
	CountdownRandomMaxTimeMs                 int    `json:"countdown_random_max_time_ms,omitempty"`
	SessionMinimumCarCount                   int    `json:"session_minimum_car_count,omitempty"`
	MaximumWaitingForPlayersDurationMs       int    `json:"maximum_waiting_for_players_duration_ms,omitempty"`
	MinimumWaitingForPlayersDurationMs       int    `json:"minimum_waiting_for_players_duration_ms,omitempty"`
	MaximumSessionWaitingForLeaderDurationMs int    `json:"maximum_session_waiting_for_leader_duration_ms,omitempty"`
	MaximumGoToBoxDurationMs                 int    `json:"maximum_go_to_box_duration_ms,omitempty"`
	StartMode                                string `json:"start_mode,omitempty"`
	StartingPosition                         int    `json:"starting_position,omitempty"`
	WaitingForPlayers                        bool   `json:"waiting_for_players,omitempty"`
}

// PenaltyTransformations holds the list of penalty escalation rules.
type PenaltyTransformations struct {
	Transformations []PenaltyTransformation `json:"transformations"`
}

// PenaltyTransformation describes how one penalty type escalates into another.
type PenaltyTransformation struct {
	Type                      string                    `json:"type"`
	TriggerType               string                    `json:"trigger_type"`
	TriggerCountdown          int                       `json:"trigger_countdown"`
	TriggeredData             PenaltyData               `json:"triggered_data"`
	OptionalClearingType      string                    `json:"optional_clearing_type"`
	OptionalClearingCountdown int                       `json:"optional_clearing_countdown"`
	PostSessionTransformation PostSessionTransformation `json:"post_session_transformation"`
}

// PostSessionTransformation holds options applied to a penalty after a session ends.
type PostSessionTransformation struct {
	AddPitlaneTime bool `json:"add_pitlane_time"`
}

// PenaltyData describes a penalty outcome.
type PenaltyData struct {
	Type            string `json:"type"`
	PenaltyTimeMs   int    `json:"penalty_time_ms"`
	LapsForClearing int    `json:"laps_for_clearing"`
}

// PenaltyInvestigations holds the list of on-track incident triggers.
type PenaltyInvestigations struct {
	Triggers []PenaltyTrigger `json:"triggers"`
}

// PenaltyTrigger represents an incident condition along with its penalty checks.
// Exactly one of the condition fields will be non-nil.
type PenaltyTrigger struct {
	RaceCarCut           *RaceCarCutCondition           `json:"race_car_cut,omitempty"`
	WrongWay             *WrongWayCondition             `json:"wrong_way,omitempty"`
	Speeding             *SpeedingCondition             `json:"speeding,omitempty"`
	WrongPositionOnStart *WrongPositionOnStartCondition `json:"wrong_position_on_start,omitempty"`
	CommandName          string                         `json:"command_name"`
	Checks               []PenaltyCheck                 `json:"checks"`
}

// RaceCarCutCondition is triggered when a car cuts the track.
type RaceCarCutCondition struct {
	TyresOut      int     `json:"tires_out"`
	WetMultiplier float64 `json:"wet_multiplier"`
}

// WrongWayCondition is triggered when a driver travels the wrong direction.
type WrongWayCondition struct {
	MinSpeed             float64 `json:"min_speed"`
	SpeedMultiplier      float64 `json:"speed_multiplier"`
	OutOfTrackMultiplier float64 `json:"out_of_track_multiplier"`
}

// SpeedingCondition is triggered when a driver exceeds a speed limit in a zone.
type SpeedingCondition struct {
	RequiredSpeed float64 `json:"required_speed"`
	CheckIfHigher bool    `json:"check_if_higher"`
}

// WrongPositionOnStartCondition is triggered when a car starts in the wrong grid position.
type WrongPositionOnStartCondition struct {
	DistanceOnGridThresholds  []Threshold `json:"distance_on_grid_thresholds"`
	DistanceOnStartThresholds []Threshold `json:"distance_on_start_thresholds"`
	RotationThresholds        []Threshold `json:"rotation_thresholds"`
}

// Threshold pairs a numeric threshold value with a severity level.
type Threshold struct {
	Threshold float64 `json:"threshold"`
	Level     int     `json:"level"`
}

// PenaltyCheck combines a penalty outcome with a weighted probability.
type PenaltyCheck struct {
	Data   PenaltyData `json:"data"`
	Weight int         `json:"weight"`
}

// CarStanding holds the per-car session summary.
type CarStanding struct {
	CarID                GUID               `json:"car_id"`
	TotalKm              float64            `json:"total_km"`
	TotalFuelLiters      float64            `json:"total_fuel_liters"`
	EnergySourceConsumed float64            `json:"energy_source_consumed"`
	EnergySourceType     string             `json:"energy_source_type"`
	StartingPosition     int                `json:"starting_position"`
	TyreTreadConsumed    map[string]float64 `json:"tire_tread_consumed"`
}

// Driver contains the personal information for a race participant.
type Driver struct {
	GUID      GUID   `json:"guid"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Nickname  string `json:"nickname"`
	PlayerID  string `json:"player_id"`
	Nation    string `json:"nation"`
}

// Lap represents a single timed lap recorded during the session.
type Lap struct {
	CarKey    GUID  `json:"car_key"`
	DriverKey GUID  `json:"driver_key"`
	Time      int   `json:"time"`
	Split     []int `json:"split"`
	Flags     int   `json:"flags"`
}

// Car contains the static metadata for a car entered in the session.
type Car struct {
	CarID                 GUID    `json:"car_id"`
	ModelDisplayName      string  `json:"model_displayname"`
	ModelMechanicalPreset string  `json:"model_mechanical_preset"`
	PerformanceIndicator  float64 `json:"performance_indicator"`
	RaceNumber            int     `json:"race_number"`
}
