//nolint:lll // tests + testdata
package acc

import (
	"strings"
	"testing"
	"unicode/utf16"
)

//nolint:gocyclo,funlen // complex, yes
func TestParseJSONMinimal(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"sessionType":"R",
		"trackName":"cota",
		"sessionIndex":2,
		"raceWeekendIndex":0,
		"metaData":"cota",
		"serverName":"vr",
		"sessionResult":{
			"bestlap":124877,
			"bestSplits":[26502,40660,57713],
			"isWetSession":false,
			"type":"Race",
			"leaderBoardLines":[
				{
					"car":{
						"carId":1006,
						"raceNumber":52,
						"carModel":32,
						"cupCategory":0,
						"carGroup":"GT3",
						"teamName":"Nitrous Oxide",
						"nationality":17,
						"carGuid":-1,
						"teamGuid":-1,
						"drivers":[
							{"firstName":"Leif","lastName":"Mille","shortName":"L. MIL","playerId":"S123"}
						]
					},
					"currentDriver":{"firstName":"Leif","lastName":"Mille","shortName":"L. MIL","playerId":"S123"},
					"currentDriverIndex":0,
					"driverTotalTimes":[5413279.5],
					"missingMandatoryPitstop":-1,
					"timing":{"lastLap":159997,"lastSplits":[36577,54983,68437],"bestLap":125425,"bestSplits":[35275,54240,35910],"totalTime":5528712,"lapCount":43,"lastSplitId":2}
				}
			]
		},
		"laps":[{"carId":1006,"driverIndex":0,"laptime":140655,"isValidForBest":false,"splits":[311927,423839,140655]}],
		"penalties":[{"carId":1006,"driverIndex":0,"reason":"PitSpeeding","penalty":"StopAndGo_30","penaltyValue":2,"violationInLap":1,"clearedInLap":5}],
		"post_race_penalties":[]
	}`)

	parsed, err := ParseJSON(payload)
	if err != nil {
		t.Fatalf("ParseJSON returned unexpected error: %v", err)
	}

	if !parsed.Session.StartTime.IsZero() {
		t.Fatalf("expected zero start time, got %v", parsed.Session.StartTime)
	}

	if parsed.Session.Track != "cota" {
		t.Fatalf("unexpected track: got %q", parsed.Session.Track)
	}

	if len(parsed.Results) != 1 {
		t.Fatalf("unexpected result count: got %d want %d", len(parsed.Results), 1)
	}

	row := parsed.Results[0]
	if row.FinPos != 1 ||
		row.CarID != "32" ||
		row.Car != "32" ||
		row.TeamID != "-1" ||
		row.DriverID != "S123" ||
		row.Name != "Leif Mille" ||
		row.CarNumber != "52" ||
		row.TotalTime != 5528712 ||
		row.FastestLapTime != 125425 ||
		row.Laps != 43 ||
		row.Incidents != 1 {

		t.Fatalf("unexpected parsed row: %+v", row)
	}
}

func TestParseJSONUTF16LEPayload(t *testing.T) {
	t.Parallel()

	jsonText := `{"sessionType":"Q","trackName":"cota","sessionIndex":1,"raceWeekendIndex":0,"metaData":"cota","serverName":"vr","sessionResult":{"bestlap":1000,"bestSplits":[1,2,3],"isWetSession":false,"type":"Q","leaderBoardLines":[{"car":{"carId":1,"raceNumber":7,"carModel":99,"cupCategory":0,"carGroup":"GT3","teamName":"Team","nationality":1,"carGuid":1,"teamGuid":2,"drivers":[{"firstName":"A","lastName":"B","shortName":"A. B","playerId":"P1"}]},"currentDriver":{"firstName":"A","lastName":"B","shortName":"A. B","playerId":"P1"},"currentDriverIndex":0,"driverTotalTimes":[1.5],"missingMandatoryPitstop":-1,"timing":{"lastLap":1000,"lastSplits":[1,2,3],"bestLap":900,"bestSplits":[1,2,3],"totalTime":12000,"lapCount":10,"lastSplitId":2}}]},"laps":[],"penalties":[],"post_race_penalties":[]}`
	utf16Payload := encodeUTF16LE(jsonText)

	parsed, err := ParseJSON(utf16Payload)
	if err != nil {
		t.Fatalf("ParseJSON returned unexpected error for utf16 payload: %v", err)
	}

	if len(parsed.Results) != 1 {
		t.Fatalf("unexpected result count: got %d want %d", len(parsed.Results), 1)
	}

	if parsed.Results[0].QualiLapTime != 900 {
		t.Fatalf("unexpected quali lap time: got %d want %d", parsed.Results[0].QualiLapTime, 900)
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

func encodeUTF16LE(s string) []byte {
	u16 := utf16.Encode([]rune(s))
	out := make([]byte, 0, len(u16)*2)
	for i := range u16 {
		out = append(out, byte(u16[i]), byte(u16[i]>>8))
	}

	return out
}
