//nolint:lll // long lines in test data
package iracing

import (
	"reflect"
	"strings"
	"testing"
	"time"

	processor "github.com/srlmgr/backend/services/importsvc/importer"
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
	if parsed.DataType != "all" {
		t.Fatalf("unexpected data type: got %q want %q", parsed.DataType, "all")
	}

	first := parsed.Results[0]
	if first.FinPos != 1 || first.CarID != "165" || first.TeamID != "1310137" ||
		first.DriverID != "1310137" || first.StartPos != 2 || first.CarNumber != "2" ||
		first.Interval != "-00.000" || first.LapsLed != 3 ||
		first.FastestLapTime != 77561 || first.Laps != 16 || first.Incidents != 0 {

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

const sampleTeamRaceCSV = `Start Time,Track,Series,Hosted Session Name
"2026-03-27T17:30:15Z","Autódromo José Carlos Pace - Grand Prix","Hosted iRacing","VR e.V. Global Endurance Event 1"

"Fin Pos","Car ID","Car","Car Class ID","Car Class","Team ID","Cust ID","Name","Start Pos","Car #","Out ID","Out","Interval","Laps Led","Qualify Time","Average Lap Time","Fastest Lap Time","Fast Lap#","Laps Comp","Inc","Club ID","Club","Max Fuel Fill%","Weight Penalty (KG)","Session Name","AI"
"1","128","Dallara P217","2523","Dallara P217","-14503","-14503","Single Driver","3","266","0","Running","-00.000","47","","1:31.248","1:28.744","56","79","12","0","","100","0","VR e.V. Global Endurance Event 1","0"
"1","128","Dallara P217","2523","Dallara P217","-14503","118646","Driver 1","3","266","0","Running","-","47","","1:31.248","1:28.744","56","79","12","0","","100","0","VR e.V. Global Endurance Event 1","0"
"2","169","Porsche 911 GT3 R (992)","4091","GT3 2025","-452610","-452610","Multiple Drivers","27","363","0","Running","-4 L","0","","1:36.705","1:33.294","53","75","14","0","","100","0","VR e.V. Global Endurance Event 1","0"
"2","169","Porsche 911 GT3 R (992)","4091","GT3 2025","-452610","1121291","Driver 2","27","363","0","Running","-","0","","1:35.189","1:33.294","53","42","8","0","","100","0","VR e.V. Global Endurance Event 1","0"
"2","169","Porsche 911 GT3 R (992)","4091","GT3 2025","-452610","720121","Driver 3","27","363","0","Running","-","0","","1:36.972","1:34.178","17","33","6","0","","100","0","VR e.V. Global Endurance Event 1","0"
`

//nolint:funlen // much data to validate
func TestParseCSVTeamRace(t *testing.T) {
	t.Parallel()

	parsed, err := ParseCSV([]byte(sampleTeamRaceCSV))
	if err != nil {
		t.Fatalf("ParseCSV returned unexpected error: %v", err)
	}
	expectedStart := time.Date(2026, 3, 27, 17, 30, 15, 0, time.UTC)
	if !parsed.Session.StartTime.Equal(expectedStart) {
		t.Fatalf("unexpected start time: got %v want %v", parsed.Session.StartTime, expectedStart)
	}
	if parsed.Session.Track != "Autódromo José Carlos Pace - Grand Prix" {
		t.Fatalf("unexpected track: got %q", parsed.Session.Track)
	}

	if len(parsed.Results) != 2 {
		t.Fatalf("unexpected result count: got %d want %d", len(parsed.Results), 2)
	}
	if parsed.DataType != "all" {
		t.Fatalf("unexpected data type: got %q want %q", parsed.DataType, "all")
	}
	first := parsed.Results[0]
	expectedFirst := &processor.ResultRow{
		FinPos:         1,
		CarID:          "128",
		Car:            "Dallara P217",
		TeamID:         "-14503",
		DriverID:       "-14503",
		Name:           "Single Driver",
		StartPos:       3,
		CarNumber:      "266",
		Interval:       "-00.000",
		LapsLed:        47,
		QualiLapTime:   -1,
		TotalTime:      7208592,
		FastestLapTime: 88744,
		Laps:           79,
		Incidents:      12,
		TeamDrivers: []*processor.TeamDriver{
			{
				DriverID:       "118646",
				Name:           "Driver 1",
				FastestLapTime: 88744,
				Laps:           79,
				Incidents:      12,
			},
		},
	}
	if !reflect.DeepEqual(first, expectedFirst) {
		t.Fatalf("unexpected first result row: got %+v want %+v", first, expectedFirst)
	}
	second := parsed.Results[1]
	expectedSecond := &processor.ResultRow{
		FinPos:         2,
		CarID:          "169",
		Car:            "Porsche 911 GT3 R (992)",
		TeamID:         "-452610",
		DriverID:       "-452610",
		Name:           "Multiple Drivers",
		StartPos:       27,
		CarNumber:      "363",
		Interval:       "-4 L",
		LapsLed:        0,
		QualiLapTime:   -1,
		TotalTime:      7252875,
		FastestLapTime: 93294,
		Laps:           75,
		Incidents:      14,

		TeamDrivers: []*processor.TeamDriver{
			{
				DriverID:       "1121291",
				Name:           "Driver 2",
				FastestLapTime: 93294,
				Laps:           42,
				Incidents:      8,
			},
			{
				DriverID:       "720121",
				Name:           "Driver 3",
				FastestLapTime: 94178,
				Laps:           33,
				Incidents:      6,
			},
		},
	}
	if !reflect.DeepEqual(second, expectedSecond) {
		t.Fatalf("unexpected second result row: got %+v want %+v", second, expectedSecond)
	}
}
