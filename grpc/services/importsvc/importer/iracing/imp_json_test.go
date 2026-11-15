//nolint:lll // long json fixtures in tests
package iracing

import (
	"context"
	"reflect"
	"testing"
	"time"

	importer "github.com/srlmgr/backend/grpc/services/importsvc/importer"
	"github.com/srlmgr/backend/support/iracing/irdata"
)

func TestJSONProcessorCollectIncidentsCanBeDisabled(t *testing.T) {
	t.Parallel()

	ctx := irdata.AddToContext(
		context.Background(),
		&irdata.IrData{},
	)
	p := &jsonProcessor{
		ctx: importer.WithoutLapDataCollection(ctx),
		data: &EventResult{
			SessionResults: []SimSession{
				{
					SimsessionTypeName: "Race",
					Results:            []Result{{CustID: 42}},
				},
			},
		},
		lapIncidents: make(incsBySession),
	}

	p.collectIncidents()

	if len(p.lapIncidents) != 0 {
		t.Fatalf("expected no lap incidents, got %v", p.lapIncidents)
	}
}

//nolint:funlen // table cases cover the heuristic boundaries and opt-in
func TestJSONProcessorBuildRacePayload_OfftrackHeuristic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		incidents        int
		events           []string
		raceSequenceNo   int
		enabledByRace    map[int]bool
		expectedOfftrack int
	}{
		{
			name:             "infers multiple offtracks from one offtrack event",
			incidents:        3,
			events:           []string{"off track"},
			enabledByRace:    map[int]bool{0: true},
			expectedOfftrack: 3,
		},
		{
			name:             "uses maximum incident values for other events",
			incidents:        11,
			events:           []string{"off track", "lost control", "car contact", "contact"},
			enabledByRace:    map[int]bool{0: true},
			expectedOfftrack: 3,
		},
		{
			name:             "does not infer negative offtracks",
			incidents:        8,
			events:           []string{"off track", "lost control", "car contact", "contact"},
			enabledByRace:    map[int]bool{0: true},
			expectedOfftrack: 1,
		},
		{
			name:             "leaves offtracks unchanged when disabled",
			incidents:        3,
			events:           []string{"off track"},
			enabledByRace:    map[int]bool{0: false},
			expectedOfftrack: 1,
		},
		{
			name:             "uses setting for the matching race sequence",
			incidents:        3,
			events:           []string{"off track"},
			raceSequenceNo:   2,
			enabledByRace:    map[int]bool{1: true, 2: false},
			expectedOfftrack: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &jsonProcessor{
				ctx: importer.WithOfftrackHeuristicByRace(
					context.Background(),
					tt.enabledByRace,
				),
				data: &EventResult{},
				lapIncidents: incsBySession{
					1: {
						{
							RefID: 42,
							Incidents: []*incidentData{
								{Events: tt.events},
							},
						},
					},
				},
			}
			payload := p.buildRacePayload(
				&SimSession{
					SimsessionNumber: 1,
					Results: []Result{
						{CustID: 42, Incidents: tt.incidents},
					},
				},
				false,
				tt.raceSequenceNo,
			)

			if got := payload.Results[0].Offtracks; got != tt.expectedOfftrack {
				t.Fatalf("unexpected offtracks: got %d want %d", got, tt.expectedOfftrack)
			}
		})
	}
}

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
							"display_name": "Driver One",
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
							"display_name": "Driver One",
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
	ctx := context.Background()
	parsed, err := ParseJSON(ctx, []byte(payload))
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
	if row.FastestLapTime != 78747 {
		t.Fatalf("unexpected fastest lap: got %d want %d", row.FastestLapTime, 78747)
	}
	if row.QualiLapTime != 78424 {
		t.Fatalf("unexpected quali lap: got %d want %d", row.QualiLapTime, 78424)
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
					"simsession_number": 1,
					"simsession_type_name": "Lone Qualifying",
					"results": [
						{
							"cust_id": 259338,
							"display_name": "Driver One",
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

	ctx := context.Background()
	parsed, err := ParseJSON(ctx, []byte(payload))
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
	if row.FinPos != 1 || row.QualiLapTime != 111962 {
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

	ctx := context.Background()
	parsed, err := ParseJSON(ctx, []byte(payload))
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
	if row.QualiLapTime != 111962 {
		t.Fatalf("unexpected quali lap: got %d want %d", row.QualiLapTime, 111962)
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

//nolint:funlen // fixture-heavy parser assertions
func TestParseJSONMultiRaceHeatAndFeature(t *testing.T) {
	t.Parallel()

	// Mirrors a real heat-race payload: QUALIFY (shared) then HEAT 1 and FEATURE, with
	// unreliable/non-sequential simsession_number values that must not be relied upon.
	payload := `{
		"type": "event_result",
		"data": {
			"max_team_drivers": 1,
			"start_time": "2026-09-03T17:00:21Z",
			"track": {"track_name": "Circuit Zandvoort", "config_name": "Grand Prix"},
			"session_results": [
				{
					"simsession_number": -4,
					"simsession_type": 5,
					"simsession_type_name": "Open Qualifying",
					"results": [
						{
							"cust_id": 111,
							"display_name": "Driver One",
							"car_id": 208,
							"car_name": "Porsche 911 Cup",
							"best_lap_time": 1000000,
							"finish_position": 0,
							"starting_position": -1,
							"laps_complete": 2,
							"laps_lead": 0,
							"incidents": 0,
							"livery": {"car_number": "1"}
						}
					]
				},
				{
					"simsession_number": -3,
					"simsession_name": "HEAT 1",
					"simsession_type": 6,
					"simsession_type_name": "Race",
					"results": [
						{
							"cust_id": 111,
							"display_name": "Driver One",
							"car_id": 208,
							"car_name": "Porsche 911 Cup",
							"best_lap_time": 950000,
							"finish_position": 0,
							"starting_position": 3,
							"laps_complete": 13,
							"laps_lead": 8,
							"incidents": 4,
							"livery": {"car_number": "1"}
						}
					]
				},
				{
					"simsession_number": 0,
					"simsession_name": "FEATURE",
					"simsession_type": 6,
					"simsession_type_name": "Race",
					"results": [
						{
							"cust_id": 111,
							"display_name": "Driver One",
							"car_id": 208,
							"car_name": "Porsche 911 Cup",
							"best_lap_time": 934159,
							"finish_position": 0,
							"starting_position": 5,
							"laps_complete": 26,
							"laps_lead": 24,
							"incidents": 2,
							"livery": {"car_number": "1"}
						}
					]
				}
			]
		}
	}`
	ctx := context.Background()
	payloads, err := ParseJSONMultiRace(ctx, []byte(payload))
	if err != nil {
		t.Fatalf("ParseJSONMultiRace returned unexpected error: %v", err)
	}

	if len(payloads) != 2 {
		t.Fatalf("unexpected race count: got %d want %d", len(payloads), 2)
	}

	heat := payloads[0]
	if heat.RaceSequenceNo != 1 {
		t.Fatalf("unexpected heat sequence no: got %d want %d", heat.RaceSequenceNo, 1)
	}
	if heat.Results[0].Laps != 13 || heat.Results[0].LapsLed != 8 {
		t.Fatalf("unexpected heat result mapping: %+v", heat.Results[0])
	}
	// quali lap time is shared across all races since one quali session applies to all heats.
	if heat.Results[0].QualiLapTime != 100000 {
		t.Fatalf("unexpected heat quali lap: got %d want %d", heat.Results[0].QualiLapTime, 100000)
	}

	feature := payloads[1]
	if feature.RaceSequenceNo != 2 {
		t.Fatalf("unexpected feature sequence no: got %d want %d", feature.RaceSequenceNo, 2)
	}
	if feature.Results[0].Laps != 26 || feature.Results[0].LapsLed != 24 {
		t.Fatalf("unexpected feature result mapping: %+v", feature.Results[0])
	}
	// StartPos must come from the feature's own starting_position, not the heat's.
	if feature.Results[0].StartPos != 6 {
		t.Fatalf("unexpected feature start pos: got %d want %d", feature.Results[0].StartPos, 6)
	}
	// Quali only sets the grid for the first race; later races must not inherit it.
	if feature.Results[0].QualiLapTime != 0 {
		t.Fatalf(
			"unexpected feature quali lap: got %d want %d",
			feature.Results[0].QualiLapTime,
			0,
		)
	}

	// ParseJSON (single-race entrypoint) must still return only the first detected race,
	// preserving behavior for callers unaware of heat races.
	single, err := ParseJSON(ctx, []byte(payload))
	if err != nil {
		t.Fatalf("ParseJSON returned unexpected error: %v", err)
	}
	if single.RaceSequenceNo != 1 || single.Results[0].Laps != 13 {
		t.Fatalf("unexpected single-race fallback: %+v", single)
	}
}
