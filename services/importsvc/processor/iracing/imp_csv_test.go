//nolint:lll // long lines in test data
package iracing

import (
	"strings"
	"testing"
	"time"
)

const sampleCSVPayload = `"Start Time","Track","Series","Season Year","Season Quarter","Rookie Season","Race Week","Strength of Field","Special Event Type"
"2026-03-22T04:30:00Z","Adelaide Street Circuit","LMP3 Trophy","2026","2","N/A","1","1296",""

"Fin Pos","Car ID","Car","Car Class ID","Car Class","Team ID","Cust ID","Name","Start Pos","Car #","Out ID","Out","Interval","Laps Led","Qualify Time","Average Lap Time","Fastest Lap Time","Fast Lap#","Laps Comp","Inc","Pts","Club Pts","Div","Club ID","Club","Old iRating","New iRating","Old License Level","Old License Sub-Level","New License Level","New License Sub-Level","Series Name","Max Fuel Fill%","Weight Penalty (KG)","Agg Pts","AI"
"1","165","Ligier JS P320","4018","Ligier JS P320","1310137","1310137","Alonso Honores","2","2","0","Running","-00.000","3","","1:18.590","1:17.561","4","16","0","77","","5","0","","1638","1714","11","375","11","393","LMP3 Trophy","100","0","77","0"
"2","165","Ligier JS P320","4018","Ligier JS P320","1324925","1324925","Zac Hutterer","1","1","0","Running","-15.549","13","","1:19.562","1:16.921","9","16","6","70","","5","0","","2059","2103","19","377","19","354","LMP3 Trophy","100","0","70","0"
`

//nolint:gocyclo // specific test
func TestParseCSV(t *testing.T) {
	t.Parallel()

	parsed, err := ParseCSV([]byte(sampleCSVPayload))
	if err != nil {
		t.Fatalf("ParseCSV returned unexpected error: %v", err)
	}

	expectedStart := time.Date(2026, 3, 22, 4, 30, 0, 0, time.UTC)
	if !parsed.Session.StartTime.Equal(expectedStart) {
		t.Fatalf("unexpected start time: got %v want %v", parsed.Session.StartTime, expectedStart)
	}
	if parsed.Session.Track != "Adelaide Street Circuit" {
		t.Fatalf("unexpected track: got %q", parsed.Session.Track)
	}

	if len(parsed.Results) != 2 {
		t.Fatalf("unexpected result count: got %d want %d", len(parsed.Results), 2)
	}

	first := parsed.Results[0]
	if first.FinPos != 1 || first.CarID != "165" || first.TeamID != "1310137" ||
		first.DriverID != "1310137" || first.StartPos != 2 || first.CarNumber != "2" ||
		first.Interval != "-00.000" || first.LapsLed != 3 ||
		first.FastestLapTime != "1:17.561" || first.Laps != 16 || first.Incidents != 0 {

		t.Fatalf("unexpected first result row: %+v", first)
	}
}

func TestParseCSVRequiresTwoSections(t *testing.T) {
	t.Parallel()

	payload := `"Start Time","Track"
"2026-03-22T04:30:00Z","Adelaide Street Circuit"`

	_, err := ParseCSV(payload)
	if err == nil {
		t.Fatal("expected error for payload without second csv section")
	}
	if !strings.Contains(err.Error(), "expected two csv sections") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCSVMissingLaps(t *testing.T) {
	t.Parallel()

	payload := `"Start Time","Track"
"2026-03-22T04:30:00Z","Adelaide Street Circuit"

"Fin Pos","Car ID","Car","Team ID","Cust ID","Name","Start Pos","Car #","Interval","Laps Led","Fastest Lap Time","Laps","Inc"
"1","165","Ligier JS P320","1310137","1310137","Alonso Honores","2","2","-00.000","3","1:17.561","16","0"`

	_, err := ParseCSV(payload)
	if err == nil {
		t.Fatal("expected error for payload without second csv section")
	}

	if !strings.Contains(err.Error(), "missing required field: Laps Comp") {
		t.Fatalf("unexpected error: %v", err)
	}
}
