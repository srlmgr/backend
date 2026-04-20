package r3e

import (
	"strings"
	"testing"
	"time"
)

//nolint:gocyclo,funlen // test
func TestParseJSONMinimal(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"StartTime": 1774801690,
		"Track": "Dubai Autodrome",
		"TrackLayout": "Grand Prix Circuit",
		"Sessions": [
			{
				"Type": "Qualify",
				"Players": [
					{
						"UserId": 42,
						"FullName": "Jane Doe",
						"CarId": 13137,
						"Car": "Lamborghini SC63",
						"Position": 1,
						"StartPosition": 1,
						"BestLapTime": 106156,
						"TotalTime": -1,
						"RaceSessionLaps": []
					}
				]
			},
			{
				"Type": "Race",
				"Players": [
					{
						"UserId": 42,
						"FullName": "Jane Doe",
						"CarId": 13137,
						"Car": "Lamborghini SC63",
						"Position": 2,
						"StartPosition": 3,
						"BestLapTime": 106396,
						"TotalTime": 4816996,
						"RaceSessionLaps": [
							{
								"Incidents": [
									{"Points": 1},
									{"Points": 2}
								]
							},
							{
								"Incidents": []
							}
						]
					}
				]
			}
		]
	}`)

	parsed, err := ParseJSON(payload)
	if err != nil {
		t.Fatalf("ParseJSON returned unexpected error: %v", err)
	}

	expectedStart := time.Unix(1774801690, 0).UTC()
	if !parsed.Session.StartTime.Equal(expectedStart) {
		t.Fatalf("unexpected start time: got %v want %v",
			parsed.Session.StartTime, expectedStart)
	}

	if parsed.Session.Track != "Dubai Autodrome - Grand Prix Circuit" {
		t.Fatalf("unexpected track: got %q", parsed.Session.Track)
	}

	if len(parsed.Results) != 1 {
		t.Fatalf("unexpected results count: got %d want %d", len(parsed.Results), 1)
	}

	row := parsed.Results[0]
	if row.FinPos != 2 ||
		row.CarID != "13137" ||
		row.DriverID != "42" ||
		row.TeamID != "42" ||
		row.Name != "Jane Doe" ||
		row.StartPos != 3 ||
		row.QualiLapTime != 106156 ||
		row.TotalTime != 4816996 ||
		row.FastestLapTime != 106396 ||
		row.Laps != 2 ||
		row.Incidents != 3 {

		t.Fatalf("unexpected parsed row: %+v", row)
	}
}

func TestParseJSONMissingRaceSession(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"StartTime": 1774801690,
		"Track": "Dubai Autodrome",
		"TrackLayout": "Grand Prix Circuit",
		"Sessions": [{"Type": "Qualify", "Players": []}]
	}`)

	_, err := ParseJSON(payload)
	if err == nil {
		t.Fatal("expected error for payload without race session")
	}

	if !strings.Contains(err.Error(), "missing required session type: race") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseJSONUnsupportedPayloadType(t *testing.T) {
	t.Parallel()

	_, err := ParseJSON(123)
	if err == nil {
		t.Fatal("expected error for unsupported payload type")
	}

	if !strings.Contains(err.Error(), "unsupported json payload type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
