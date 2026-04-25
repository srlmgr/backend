//nolint:lll // long json fixtures in tests
package iracing

import (
	"reflect"
	"testing"
	"time"
)

//nolint:funlen // fixture-heavy parser assertions
func TestParseJSONRaceAndQualifying(t *testing.T) {
	t.Parallel()

	payload := `{
		"type": "event_result",
		"data": {
			"max_team_drivers": 1,
			"start_time": "2026-04-16T17:00:23Z",
			"track": {"track_name": "Long Beach Street Circuit", "config_name": "N/A"},
			"session_results": [
				{
					"simsession_type": 5,
					"simsession_type_name": "Open Qualifying",
					"results": [
						{
							"cust_id": 118646,
							"display_name": "Christoph Klawiter",
							"car_id": 208,
							"car_name": "Porsche 911 Cup",
							"best_lap_time": 784241,
							"finish_position": 0,
							"starting_position": 0,
							"laps_complete": 2,
							"laps_lead": 0,
							"incidents": 0,
							"livery": {"car_number": "8"}
						}
					]
				},
				{
					"simsession_type": 6,
					"simsession_type_name": "Race",
					"results": [
						{
							"cust_id": 118646,
							"display_name": "Christoph Klawiter",
							"car_id": 208,
							"car_name": "Porsche 911 Cup",
							"best_lap_time": 787479,
							"finish_position": 0,
							"starting_position": 1,
							"laps_complete": 23,
							"laps_lead": 23,
							"incidents": 0,
							"livery": {"car_number": "8"}
						}
					]
				}
			]
		}
	}`

	parsed, err := ParseJSON([]byte(payload))
	if err != nil {
		t.Fatalf("ParseJSON returned unexpected error: %v", err)
	}

	if parsed.DataType != "all" {
		t.Fatalf("unexpected data type: got %q want %q", parsed.DataType, "all")
	}

	expectedStart := time.Date(2026, 4, 16, 17, 0, 23, 0, time.UTC)
	if !parsed.Session.StartTime.Equal(expectedStart) {
		t.Fatalf("unexpected start time: got %v want %v", parsed.Session.StartTime, expectedStart)
	}
	if parsed.Session.Track != "Long Beach Street Circuit" {
		t.Fatalf("unexpected track: got %q", parsed.Session.Track)
	}

	if len(parsed.Results) != 1 {
		t.Fatalf("unexpected result count: got %d want %d", len(parsed.Results), 1)
	}
	row := parsed.Results[0]
	if row.DriverID != "118646" {
		t.Fatalf("unexpected ids: got driver=%q ", row.DriverID)
	}
	if row.FinPos != 1 || row.StartPos != 2 || row.Laps != 23 || row.LapsLed != 23 {
		t.Fatalf("unexpected race mapping: %+v", row)
	}
	if row.FastestLapTime != 787479 {
		t.Fatalf("unexpected fastest lap: got %d want %d", row.FastestLapTime, 787479)
	}
	if row.QualiLapTime != 784241 {
		t.Fatalf("unexpected quali lap: got %d want %d", row.QualiLapTime, 784241)
	}
}

//nolint:funlen // fixture-heavy parser assertions
func TestParseJSONQualifyingOnlyType6(t *testing.T) {
	t.Parallel()

	payload := `{
		"type": "event_result",
		"data": {
			"max_team_drivers": 1,
			"start_time": "2026-04-24T16:30:11Z",
			"track": {"track_name": "Miami International Autodrome", "config_name": "Grand Prix"},
			"session_results": [
				{
					"simsession_type": 6,
					"simsession_type_name": "Lone Qualifying",
					"results": [
						{
							"cust_id": 259338,
							"display_name": "Marc Landskron",
							"car_id": 169,
							"car_name": "Porsche 911 GT3 R",
							"best_lap_time": 1119626,
							"finish_position": 0,
							"starting_position": 0,
							"laps_complete": 3,
							"laps_lead": 0,
							"incidents": 1,
							"livery": {"car_number": "363"}
						}
					]
				}
			]
		}
	}`

	parsed, err := ParseJSON(payload)
	if err != nil {
		t.Fatalf("ParseJSON returned unexpected error: %v", err)
	}

	if parsed.DataType != "quali" {
		t.Fatalf("unexpected data type: got %q want %q", parsed.DataType, "quali")
	}
	if parsed.Session.Track != "Miami International Autodrome - Grand Prix" {
		t.Fatalf("unexpected track: got %q", parsed.Session.Track)
	}
	if len(parsed.Results) != 1 {
		t.Fatalf("unexpected result count: got %d want %d", len(parsed.Results), 1)
	}
	row := parsed.Results[0]
	if row.FinPos != 1 || row.QualiLapTime != 1119626 {
		t.Fatalf("unexpected quali mapping: %+v", row)
	}
}

//nolint:funlen // fixture-heavy parser assertions
func TestParseJSONTeamDriversFromAllSessions(t *testing.T) {
	t.Parallel()

	payload := `{
		"type": "event_result",
		"data": {
			"max_team_drivers": 15,
			"start_time": "2026-04-24T16:30:11Z",
			"track": {"track_name": "Miami International Autodrome", "config_name": "Grand Prix"},
			"session_results": [
				{
					"simsession_type": 3,
					"simsession_type_name": "Open Practice",
					"results": [
						{
							"team_id": -385239,
							"display_name": "Austrian Simracers VRGES",
							"car_id": 169,
							"car_name": "Porsche 911 GT3 R",
							"best_lap_time": 1121000,
							"finish_position": 0,
							"starting_position": 0,
							"laps_complete": 4,
							"laps_lead": 0,
							"incidents": 1,
							"livery": {"car_number": "51"},
							"driver_results": [
								{"cust_id": 333, "display_name": "Driver Three", "best_lap_time": 1121000, "laps_complete": 4, "incidents": 1}
							]
						}
					]
				},
				{
					"simsession_type": 4,
					"simsession_type_name": "Lone Qualifying",
					"results": [
						{
							"team_id": -385239,
							"display_name": "Austrian Simracers VRGES",
							"car_id": 169,
							"car_name": "Porsche 911 GT3 R",
							"best_lap_time": 1119626,
							"finish_position": 0,
							"starting_position": 0,
							"laps_complete": 3,
							"laps_lead": 0,
							"incidents": 0,
							"livery": {"car_number": "51"},
							"driver_results": [
								{"cust_id": 111, "display_name": "Driver One", "best_lap_time": 1119626, "laps_complete": 3, "incidents": 0}
							]
						}
					]
				},
				{
					"simsession_type": 6,
					"simsession_type_name": "Race",
					"results": [
						{
							"team_id": -385239,
							"display_name": "Austrian Simracers VRGES",
							"car_id": 169,
							"car_name": "Porsche 911 GT3 R",
							"best_lap_time": 1130000,
							"finish_position": 1,
							"starting_position": 3,
							"laps_complete": 75,
							"laps_lead": 0,
							"incidents": 14,
							"livery": {"car_number": "51"},
							"driver_results": [
								{"cust_id": 111, "display_name": "Driver One", "best_lap_time": 1130000, "laps_complete": 42, "incidents": 8},
								{"cust_id": 222, "display_name": "Driver Two", "best_lap_time": 1132000, "laps_complete": 33, "incidents": 6}
							]
						}
					]
				}
			]
		}
	}`

	parsed, err := ParseJSON([]byte(payload))
	if err != nil {
		t.Fatalf("ParseJSON returned unexpected error: %v", err)
	}

	if parsed.DataType != "all" {
		t.Fatalf("unexpected data type: got %q want %q", parsed.DataType, "all")
	}
	if len(parsed.Results) != 1 {
		t.Fatalf("unexpected result count: got %d want %d", len(parsed.Results), 1)
	}

	row := parsed.Results[0]
	if row.TeamID != "-385239" {
		t.Fatalf("unexpected team ids: got team=%q ", row.TeamID)
	}
	if row.QualiLapTime != 1119626 {
		t.Fatalf("unexpected quali lap: got %d want %d", row.QualiLapTime, 1119626)
	}

	wantDrivers := []string{"111", "222", "333"}
	gotDrivers := make([]string, 0, len(row.TeamDrivers))
	for i := range row.TeamDrivers {
		gotDrivers = append(gotDrivers, row.TeamDrivers[i].DriverID)
	}
	if !reflect.DeepEqual(gotDrivers, wantDrivers) {
		t.Fatalf("unexpected team driver ids: got %v want %v", gotDrivers, wantDrivers)
	}
}
