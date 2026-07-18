//nolint:lll,funlen // test setup
package acevo

import (
	"os"
	"strings"
	"testing"
)

// minimalQualiJSON is a stripped-down qualify session with two drivers.
const minimalQualiJSON = `{
  "server_name": "test-server",
  "server_id": "",
  "server_ip": "",
  "season_guid": "aaaa-bbbb",
  "session_name": "Race Weekend",
  "session_type": "Qualify",
  "track_name": "Monza",
  "track_layout_name": "GP",
  "championship_id": {"a": "1", "b": "2"},
  "event_index": 0,
  "session_index": 0,
  "is_static": false,
  "is_completed": true,
  "specialization": {
    "@type": "type.googleapis.com/TimeAttack.Specialization",
    "base": {"session_duration_ms": 600000, "session_laps": 0,
      "maximum_session_overtime_duration_ms": 180000,
      "maximum_session_overtime_before_next_session": 60000,
      "intro_music": true, "end_music": true, "end_replay_type": "None"},
    "base_file": "", "base_source": "Custom",
    "penalty_transformations": {"transformations": []},
    "penalty_transformations_file": "EA1", "penalty_transformations_source": "Custom",
    "penalty_investigations": {"triggers": []},
    "penalty_investigations_file": "EA1", "penalty_investigations_source": "Custom",
    "rules": []
  },
  "car_standings": [],
  "driver_standings": [
    {"a": "100", "b": "200"},
    {"a": "101", "b": "201"}
  ],
  "time_standings": [108834, 109125],
  "drivers": [
    {"guid": {"a": "100", "b": "200"}, "first_name": "Alice", "last_name": "Smith",
      "nickname": "", "player_id": "STEAM100", "nation": "DEU"},
    {"guid": {"a": "101", "b": "201"}, "first_name": "Bob", "last_name": "Jones",
      "nickname": "", "player_id": "STEAM101", "nation": "ITA"}
  ],
  "laps": [
    {"car_key": {"a": "10", "b": "20"}, "driver_key": {"a": "100", "b": "200"}, "time": 108834, "split": [35000, 37000, 36834], "flags": 2},
    {"car_key": {"a": "10", "b": "20"}, "driver_key": {"a": "100", "b": "200"}, "time": 115000, "split": [36000, 40000, 39000], "flags": 1},
    {"car_key": {"a": "11", "b": "21"}, "driver_key": {"a": "101", "b": "201"}, "time": 109125, "split": [35100, 37025, 37000], "flags": 2}
  ],
  "car_points": [],
  "driver_points": [],
  "cars": [
    {"car_id": {"a": "10", "b": "20"}, "model_displayname": "Porsche 911 GT3 Cup (992)", "model_mechanical_preset": "tbd.", "performance_indicator": 0, "race_number": 2},
    {"car_id": {"a": "11", "b": "21"}, "model_displayname": "Porsche 911 GT3 Cup (992)", "model_mechanical_preset": "tbd.", "performance_indicator": 0, "race_number": 7}
  ]
}`

// minimalRaceJSON is a stripped-down race session with two drivers.
const minimalRaceJSON = `{
  "server_name": "test-server",
  "server_id": "",
  "server_ip": "",
  "season_guid": "aaaa-bbbb",
  "session_name": "Race Weekend",
  "session_type": "Race",
  "track_name": "Monza",
  "track_layout_name": "GP",
  "championship_id": {"a": "1", "b": "2"},
  "event_index": 0,
  "session_index": 0,
  "is_static": false,
  "is_completed": true,
  "specialization": {
    "@type": "type.googleapis.com/InstantRace.Specialization",
    "base": {"countdown_time_ms": 5000, "session_duration_ms": 1800000,
      "session_laps": 0, "maximum_session_overtime_duration_ms": 289000,
      "maximum_session_overtime_before_next_session": 60000,
      "intro_music": true, "end_music": true, "end_replay_type": "None"},
    "base_file": "", "base_source": "Custom",
    "penalty_transformations": {"transformations": []},
    "penalty_transformations_file": "EA1", "penalty_transformations_source": "Custom",
    "penalty_investigations": {"triggers": []},
    "penalty_investigations_file": "EA1", "penalty_investigations_source": "Custom",
    "rules": [], "allowed_car_table_keys": [], "allowed_car_keys": []
  },
  "car_standings": [
    {"car_id": {"a": "10", "b": "20"}, "total_km": 50.0, "total_fuel_liters": 10.0,
      "energy_source_consumed": 10.0, "energy_source_type": "FuelLiter",
      "starting_position": 0, "tire_tread_consumed": {}},
    {"car_id": {"a": "11", "b": "21"}, "total_km": 48.0, "total_fuel_liters": 9.5,
      "energy_source_consumed": 9.5, "energy_source_type": "FuelLiter",
      "starting_position": 1, "tire_tread_consumed": {}}
  ],
  "driver_standings": [
    {"a": "100", "b": "200"},
    {"a": "101", "b": "201"}
  ],
  "time_standings": [1871206, 1872992],
  "drivers": [
    {"guid": {"a": "100", "b": "200"}, "first_name": "Alice", "last_name": "Smith",
      "nickname": "", "player_id": "STEAM100", "nation": "DEU"},
    {"guid": {"a": "101", "b": "201"}, "first_name": "Bob", "last_name": "Jones",
      "nickname": "", "player_id": "STEAM101", "nation": "ITA"}
  ],
  "laps": [
    {"car_key": {"a": "10", "b": "20"}, "driver_key": {"a": "100", "b": "200"}, "time": 113000, "split": [37000, 38000, 38000], "flags": 1},
    {"car_key": {"a": "10", "b": "20"}, "driver_key": {"a": "100", "b": "200"}, "time": 109500, "split": [35000, 37500, 37000], "flags": 2},
    {"car_key": {"a": "10", "b": "20"}, "driver_key": {"a": "100", "b": "200"}, "time": 109200, "split": [34900, 37300, 37000], "flags": 2},
    {"car_key": {"a": "11", "b": "21"}, "driver_key": {"a": "101", "b": "201"}, "time": 114000, "split": [38000, 38000, 38000], "flags": 1},
    {"car_key": {"a": "11", "b": "21"}, "driver_key": {"a": "101", "b": "201"}, "time": 110000, "split": [36000, 37000, 37000], "flags": 2},
    {"car_key": {"a": "11", "b": "21"}, "driver_key": {"a": "101", "b": "201"}, "time": 109800, "split": [35800, 37000, 37000], "flags": 2}
  ],
  "car_points": [],
  "driver_points": [],
  "cars": [
    {"car_id": {"a": "10", "b": "20"}, "model_displayname": "Porsche 911 GT3 Cup (992)", "model_mechanical_preset": "tbd.", "performance_indicator": 0, "race_number": 2},
    {"car_id": {"a": "11", "b": "21"}, "model_displayname": "Porsche 911 GT3 Cup (992)", "model_mechanical_preset": "tbd.", "performance_indicator": 0, "race_number": 7}
  ]
}`

func TestParseJSON_Quali(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSON([]byte(minimalQualiJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.DataType != "quali" {
		t.Fatalf("expected data type %q, got %q", "quali", parsed.DataType)
	}
	if parsed.Session.Track != "Monza" {
		t.Fatalf("expected track %q, got %q", "Monza", parsed.Session.Track)
	}
	if len(parsed.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(parsed.Results))
	}

	p1 := parsed.Results[0]
	if p1.StartPos != 1 {
		t.Errorf("expected StartPos 1, got %d", p1.StartPos)
	}
	if p1.QualiLapTime != 108834 {
		t.Errorf("expected QualiLapTime 108834, got %d", p1.QualiLapTime)
	}
	if p1.DriverID != "STEAM100" {
		t.Errorf("expected DriverID %q, got %q", "STEAM100", p1.DriverID)
	}
	if p1.Name != "Alice Smith" {
		t.Errorf("expected Name %q, got %q", "Alice Smith", p1.Name)
	}
	if p1.CarNumber != "2" {
		t.Errorf("expected CarNumber %q, got %q", "2", p1.CarNumber)
	}
	if p1.FinPos != 0 {
		t.Errorf("expected FinPos 0 for quali, got %d", p1.FinPos)
	}

	p2 := parsed.Results[1]
	if p2.StartPos != 2 {
		t.Errorf("expected StartPos 2, got %d", p2.StartPos)
	}
	if p2.QualiLapTime != 109125 {
		t.Errorf("expected QualiLapTime 109125, got %d", p2.QualiLapTime)
	}
	if p2.DriverID != "STEAM101" {
		t.Errorf("expected DriverID %q, got %q", "STEAM101", p2.DriverID)
	}
	if p2.CarNumber != "7" {
		t.Errorf("expected CarNumber %q, got %q", "7", p2.CarNumber)
	}
}

func TestParseJSON_Race(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSON([]byte(minimalRaceJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.DataType != "race" {
		t.Fatalf("expected data type %q, got %q", "race", parsed.DataType)
	}
	if len(parsed.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(parsed.Results))
	}

	p1 := parsed.Results[0]
	if p1.FinPos != 1 {
		t.Errorf("expected FinPos 1, got %d", p1.FinPos)
	}
	if p1.TotalTime != 1871206 {
		t.Errorf("expected TotalTime 1871206, got %d", p1.TotalTime)
	}
	if p1.Laps != 3 {
		t.Errorf("expected Laps 3, got %d", p1.Laps)
	}
	// fastest clean lap (flags==2) is 109200
	if p1.FastestLapTime != 109200 {
		t.Errorf("expected FastestLapTime 109200, got %d", p1.FastestLapTime)
	}
	if p1.DriverID != "STEAM100" {
		t.Errorf("expected DriverID %q, got %q", "STEAM100", p1.DriverID)
	}
	if p1.CarNumber != "2" {
		t.Errorf("expected CarNumber %q, got %q", "2", p1.CarNumber)
	}
	if p1.StartPos != 0 {
		t.Errorf("expected StartPos 0 for race, got %d", p1.StartPos)
	}

	p2 := parsed.Results[1]
	if p2.FinPos != 2 {
		t.Errorf("expected FinPos 2, got %d", p2.FinPos)
	}
	if p2.TotalTime != 1872992 {
		t.Errorf("expected TotalTime 1872992, got %d", p2.TotalTime)
	}
	if p2.FastestLapTime != 109800 {
		t.Errorf("expected FastestLapTime 109800, got %d", p2.FastestLapTime)
	}
}

func TestParseJSON_StringPayload(t *testing.T) {
	t.Parallel()

	parsed, err := ParseJSON(minimalQualiJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.DataType != "quali" {
		t.Fatalf("expected data type %q, got %q", "quali", parsed.DataType)
	}
}

func TestParseJSON_SessionPayload(t *testing.T) {
	t.Parallel()

	s := &Session{
		SessionType: "Race",
		TrackName:   "Spa",
	}
	parsed, err := ParseJSON(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.DataType != "race" {
		t.Fatalf("expected data type %q, got %q", "race", parsed.DataType)
	}
	if parsed.Session.Track != "Spa" {
		t.Fatalf("expected track %q, got %q", "Spa", parsed.Session.Track)
	}
}

func TestParseJSON_NilSessionPayload(t *testing.T) {
	t.Parallel()

	_, err := ParseJSON((*Session)(nil))
	if err == nil {
		t.Fatal("expected error for nil *Session, got nil")
	}
}

func TestParseJSON_UnsupportedPayloadType(t *testing.T) {
	t.Parallel()

	_, err := ParseJSON(12345)
	if err == nil {
		t.Fatal("expected error for unsupported payload type")
	}
	if !strings.Contains(err.Error(), "unsupported json payload type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseJSON_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseJSON([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseJSON_UnknownSessionType(t *testing.T) {
	t.Parallel()

	payload := `{"session_type":"Practice","track_name":"Spa","driver_standings":[],"time_standings":[],"drivers":[],"laps":[],"cars":[]}`
	parsed, err := ParseJSON([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.DataType != "all" {
		t.Fatalf("expected data type %q for unknown session type, got %q", "all", parsed.DataType)
	}
}

func TestParseJSON_RealQualiFile(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../../../tmp/PWC_T02_Monza_Q1.json")
	if err != nil {
		t.Skipf("test file not available: %v", err)
	}

	parsed, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("unexpected error parsing real quali file: %v", err)
	}

	if parsed.DataType != "quali" {
		t.Fatalf("expected data type %q, got %q", "quali", parsed.DataType)
	}
	if parsed.Session.Track != "Monza" {
		t.Fatalf("expected track %q, got %q", "Monza", parsed.Session.Track)
	}
	// 14 drivers completed at least one lap; one driver has time=0 (no valid lap)
	if len(parsed.Results) == 0 {
		t.Fatal("expected non-empty results")
	}

	p1 := parsed.Results[0]
	if p1.StartPos != 1 {
		t.Errorf("expected pole position StartPos 1, got %d", p1.StartPos)
	}
	// best lap in the file is 108834 ms
	if p1.QualiLapTime != 108834 {
		t.Errorf("expected pole lap time 108834, got %d", p1.QualiLapTime)
	}
}

func TestParseJSON_RealRaceFile(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../../../tmp/PWC_T02_Monza_R1.json")
	if err != nil {
		t.Skipf("test file not available: %v", err)
	}

	parsed, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("unexpected error parsing real race file: %v", err)
	}

	if parsed.DataType != "race" {
		t.Fatalf("expected data type %q, got %q", "race", parsed.DataType)
	}
	if parsed.Session.Track != "Monza" {
		t.Fatalf("expected track %q, got %q", "Monza", parsed.Session.Track)
	}
	if len(parsed.Results) == 0 {
		t.Fatal("expected non-empty results")
	}

	p1 := parsed.Results[0]
	if p1.FinPos != 1 {
		t.Errorf("expected FinPos 1, got %d", p1.FinPos)
	}
	// winner total time from time_standings[0]
	if p1.TotalTime != 1871206 {
		t.Errorf("expected TotalTime 1871206, got %d", p1.TotalTime)
	}
	if p1.Laps == 0 {
		t.Error("expected non-zero lap count for race winner")
	}
	if p1.FastestLapTime == 0 {
		t.Error("expected non-zero fastest lap time for race winner")
	}
}
