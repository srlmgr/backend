//nolint:lll,funlen,gocyclo // test setup
package lmu

import (
	"os"
	"testing"

	processor "github.com/srlmgr/backend/services/importsvc/importer"
)

const minimalRaceXML = `<?xml version="1.0" encoding="utf-8"?>
<rFactorXML version="1.0">
	<RaceResults>
		<Setting>Multiplayer</Setting>
		<ServerName></ServerName>
		<TrackVenue>Lusail International Circuit</TrackVenue>
		<TrackCourse>Lusail International Circuit</TrackCourse>
		<TrackEvent>Qatar 1812KM</TrackEvent>
		<DateTime>1773423406</DateTime>
		<TimeString>2026/03/13 18:36:46</TimeString>
		<TrackLength>5405.4</TrackLength>
		<GameVersion>1.2000</GameVersion>
		<Race>
			<DateTime>1773430522</DateTime>
			<TimeString>2026/03/13 20:35:22</TimeString>
			<Laps>2147483647</Laps>
			<Minutes>120</Minutes>
			<Stream>
				<Penalty Driver="Driver One" ID="1" Penalty="Drive Thru" Time="0" Laps="0" Reason="Speeding" et="300.0">Driver One received Drive Thru penalty</Penalty>
				<Penalty Driver="Driver One" ID="1" Penalty="Stop and Go" Time="0" Laps="0" Reason="Cutting" et="500.0">Driver One received Stop and Go penalty</Penalty>
				<Penalty et="600.0">Driver One served Drive Thru penalty</Penalty>
			</Stream>
			<Driver>
				<Name>Driver One</Name>
				<Connected>1</Connected>
				<VehFile>test.VEH</VehFile>
				<UpgradeCode>00000000 00000000 00000000 00000000</UpgradeCode>
				<VehName>Test Car One</VehName>
				<Category>WEC 2025, Hypercar, Ferrari 499P</Category>
				<CarType>Ferrari 499P</CarType>
				<CarClass>Hyper</CarClass>
				<CarNumber>50</CarNumber>
				<TeamName>Test Team A</TeamName>
				<isPlayer>1</isPlayer>
				<ServerScored>1</ServerScored>
				<GridPos>2</GridPos>
				<Position>2</Position>
				<ClassGridPos>2</ClassGridPos>
				<ClassPosition>1</ClassPosition>
				<LapRankIncludingDiscos>1</LapRankIncludingDiscos>
				<Lap num="1" p="2" et="110.500" s1="25.0" s2="50.0" s3="35.5" topspeed="280.0" fuel="0.100" fuelUsed="0.020" ve="0.950" veUsed="0.050" twfl="0.990" twfr="0.990" twrl="0.990" twrr="0.990" fcompound="0,Medium" rcompound="0,Medium" FL="0,Medium" FR="0,Medium" RL="0,Medium" RR="0,Medium">110.5000</Lap>
				<BestLapTime>110.500</BestLapTime>
				<FinishTime>7200.000</FinishTime>
				<Laps>65</Laps>
				<Pitstops>2</Pitstops>
				<FinishStatus>Finished Normally</FinishStatus>
				<ControlAndAids startLap="1" endLap="65">PlayerControl,Clutch,AutoBlip</ControlAndAids>
			</Driver>
			<Driver>
				<Name>Driver Two</Name>
				<Connected>1</Connected>
				<VehFile>test2.VEH</VehFile>
				<UpgradeCode>00000000 00000000 00000000 00000000</UpgradeCode>
				<VehName>Test Car Two</VehName>
				<Category>WEC 2025, Hypercar, Toyota GR010</Category>
				<CarType>Toyota GR010</CarType>
				<CarClass>Hyper</CarClass>
				<CarNumber>7</CarNumber>
				<TeamName>Test Team B</TeamName>
				<isPlayer>0</isPlayer>
				<ServerScored>1</ServerScored>
				<GridPos>1</GridPos>
				<Position>1</Position>
				<ClassGridPos>1</ClassGridPos>
				<ClassPosition>2</ClassPosition>
				<LapRankIncludingDiscos>2</LapRankIncludingDiscos>
				<Lap num="1" p="1" et="112.000" s1="26.0" s2="51.0" s3="35.0" topspeed="279.0" fuel="0.100" fuelUsed="0.020" ve="0.950" veUsed="0.050" twfl="0.990" twfr="0.990" twrl="0.990" twrr="0.990" fcompound="0,Medium" rcompound="0,Medium" FL="0,Medium" FR="0,Medium" RL="0,Medium" RR="0,Medium">112.0000</Lap>
				<BestLapTime>112.000</BestLapTime>
				<FinishTime>7250.000</FinishTime>
				<Laps>65</Laps>
				<Pitstops>2</Pitstops>
				<FinishStatus>Finished Normally</FinishStatus>
				<ControlAndAids startLap="1" endLap="65">PlayerControl</ControlAndAids>
			</Driver>
		</Race>
	</RaceResults>
</rFactorXML>`

const minimalQualifyXML = `<?xml version="1.0" encoding="utf-8"?>
<rFactorXML version="1.0">
	<RaceResults>
		<Setting>Multiplayer</Setting>
		<ServerName></ServerName>
		<TrackVenue>Lusail International Circuit</TrackVenue>
		<TrackCourse>Lusail International Circuit</TrackCourse>
		<TrackEvent>Qatar 1812KM</TrackEvent>
		<DateTime>1773423406</DateTime>
		<TimeString>2026/03/13 18:36:46</TimeString>
		<TrackLength>5405.4</TrackLength>
		<GameVersion>1.2000</GameVersion>
		<Qualify>
			<DateTime>1773429498</DateTime>
			<TimeString>2026/03/13 20:18:18</TimeString>
			<Laps>2147483647</Laps>
			<Minutes>15</Minutes>
			<Stream>
				<TrackLimits Driver="Driver One" ID="1" Lap="0" WarningPoints="0" CurrentPoints="0" Resolution="7" et="52.8">No Further Action</TrackLimits>
			</Stream>
			<Driver>
				<Name>Driver One</Name>
				<Connected>1</Connected>
				<VehFile>test.VEH</VehFile>
				<UpgradeCode>00000000 00000000 00000000 00000000</UpgradeCode>
				<VehName>Test Car One</VehName>
				<Category>WEC 2025, Hypercar, Ferrari 499P</Category>
				<CarType>Ferrari 499P</CarType>
				<CarClass>Hyper</CarClass>
				<CarNumber>50</CarNumber>
				<TeamName>Test Team A</TeamName>
				<isPlayer>1</isPlayer>
				<ServerScored>1</ServerScored>
				<Position>1</Position>
				<ClassPosition>1</ClassPosition>
				<LapRankIncludingDiscos>5</LapRankIncludingDiscos>
				<Lap num="1" p="1" et="--.---" topspeed="254.95" fuel="0.137" fuelUsed="0.626" ve="0.976" veUsed="0.024" twfl="0.992" twfr="0.996" twrl="0.992" twrr="0.992" fcompound="0,Medium" rcompound="0,Medium" FL="0,Medium" FR="0,Medium" RL="0,Medium" RR="0,Medium">--.----</Lap>
				<Lap num="2" p="1" et="100.357" s1="26.1" s2="42.4" s3="31.8" topspeed="302.70" fuel="0.114" fuelUsed="0.024" ve="0.945" veUsed="0.031" twfl="0.973" twfr="0.984" twrl="0.980" twrr="0.980" fcompound="0,Medium" rcompound="0,Medium" FL="0,Medium" FR="0,Medium" RL="0,Medium" RR="0,Medium">100.3568</Lap>
				<BestLapTime>100.3568</BestLapTime>
				<Laps>7</Laps>
				<Pitstops>0</Pitstops>
				<FinishStatus>Finished Normally</FinishStatus>
				<ControlAndAids startLap="1" endLap="7">PlayerControl,Clutch,AutoBlip</ControlAndAids>
			</Driver>
			<Driver>
				<Name>Driver Two</Name>
				<Connected>1</Connected>
				<VehFile>test2.VEH</VehFile>
				<UpgradeCode>00000000 00000000 00000000 00000000</UpgradeCode>
				<VehName>Test Car Two</VehName>
				<Category>WEC 2025, Hypercar, Toyota GR010</Category>
				<CarType>Toyota GR010</CarType>
				<CarClass>Hyper</CarClass>
				<CarNumber>7</CarNumber>
				<TeamName>Test Team B</TeamName>
				<isPlayer>0</isPlayer>
				<ServerScored>1</ServerScored>
				<Position>2</Position>
				<ClassPosition>2</ClassPosition>
				<LapRankIncludingDiscos>5</LapRankIncludingDiscos>
				<Lap num="1" p="2" et="101.815" s1="26.5" s2="43.1" s3="32.2" topspeed="300.00" fuel="0.110" fuelUsed="0.022" ve="0.940" veUsed="0.030" twfl="0.970" twfr="0.982" twrl="0.978" twrr="0.978" fcompound="0,Medium" rcompound="0,Medium" FL="0,Medium" FR="0,Medium" RL="0,Medium" RR="0,Medium">101.8151</Lap>
				<BestLapTime>101.8151</BestLapTime>
				<Laps>4</Laps>
				<Pitstops>0</Pitstops>
				<FinishStatus>Finished Normally</FinishStatus>
				<ControlAndAids startLap="1" endLap="4">PlayerControl</ControlAndAids>
			</Driver>
		</Qualify>
	</RaceResults>
</rFactorXML>`

func TestParseXMLRace(t *testing.T) {
	t.Parallel()

	parsed, err := ParseXML([]byte(minimalRaceXML))
	if err != nil {
		t.Fatalf("ParseXML returned unexpected error: %v", err)
	}

	if parsed.DataType != processor.ImportDataRace {
		t.Fatalf("unexpected DataType: got %q want %q", parsed.DataType, processor.ImportDataRace)
	}

	if parsed.Session.Track != "Lusail International Circuit" {
		t.Fatalf("unexpected Track: got %q", parsed.Session.Track)
	}

	if parsed.Session.StartTime.IsZero() {
		t.Fatal("expected non-zero StartTime")
	}

	if len(parsed.Results) != 2 {
		t.Fatalf("unexpected result count: got %d want 2", len(parsed.Results))
	}

	r1 := parsed.Results[0]
	// note: test data is setup by design: entry 1 is pos 2, entry 2 is pos 1
	if r1.FinPos != 2 {
		t.Errorf("r1.FinPos: got %d want 1", r1.FinPos)
	}
	// is not set for race sessions, should default to zero
	if r1.StartPos != 0 {
		t.Errorf("r1.StartPos: got %d want 0", r1.StartPos)
	}
	if r1.Name != "Driver One" {
		t.Errorf("r1.Name: got %q want %q", r1.Name, "Driver One")
	}
	if r1.DriverID != "Driver One" {
		t.Errorf("r1.DriverID: got %q want %q", r1.DriverID, "Driver One")
	}
	if r1.Car != "Ferrari 499P" {
		t.Errorf("r1.Car: got %q want %q", r1.Car, "Ferrari 499P")
	}
	if r1.CarNumber != "50" {
		t.Errorf("r1.CarNumber: got %q want %q", r1.CarNumber, "50")
	}
	if r1.TeamID != "Test Team A" {
		t.Errorf("r1.TeamID: got %q want %q", r1.TeamID, "Test Team A")
	}
	if r1.FastestLapTime != 110500 {
		t.Errorf("r1.FastestLapTime: got %d want 110500", r1.FastestLapTime)
	}
	if r1.TotalTime != 7200000 {
		t.Errorf("r1.TotalTime: got %d want 7200000", r1.TotalTime)
	}
	if r1.Laps != 65 {
		t.Errorf("r1.Laps: got %d want 65", r1.Laps)
	}
	// 2 issued penalties; served notification should not be counted
	if r1.Incidents != 2 {
		t.Errorf("r1.Incidents: got %d want 2", r1.Incidents)
	}

	r2 := parsed.Results[1]
	if r2.FinPos != 1 {
		t.Errorf("r2.FinPos: got %d want 1", r2.FinPos)
	}
	if r2.StartPos != 0 {
		t.Errorf("r2.StartPos: got %d want 0", r2.StartPos)
	}
	if r2.Incidents != 0 {
		t.Errorf("r2.Incidents: got %d want 0", r2.Incidents)
	}
}

func TestParseXMLQualify(t *testing.T) {
	t.Parallel()

	parsed, err := ParseXML([]byte(minimalQualifyXML))
	if err != nil {
		t.Fatalf("ParseXML returned unexpected error: %v", err)
	}

	if parsed.DataType != processor.ImportDataQuali {
		t.Fatalf("unexpected DataType: got %q want %q", parsed.DataType, processor.ImportDataQuali)
	}

	if parsed.Session.Track != "Lusail International Circuit" {
		t.Fatalf("unexpected Track: got %q", parsed.Session.Track)
	}

	if len(parsed.Results) != 2 {
		t.Fatalf("unexpected result count: got %d want 2", len(parsed.Results))
	}

	r1 := parsed.Results[0]
	if r1.StartPos != 1 {
		t.Errorf("r1.StartPos: got %d want 1", r1.StartPos)
	}
	if r1.Name != "Driver One" {
		t.Errorf("r1.Name: got %q want %q", r1.Name, "Driver One")
	}
	if r1.Car != "Ferrari 499P" {
		t.Errorf("r1.Car: got %q want %q", r1.Car, "Ferrari 499P")
	}
	if r1.CarNumber != "50" {
		t.Errorf("r1.CarNumber: got %q want %q", r1.CarNumber, "50")
	}
	// 100.3568s → 100357ms (rounded)
	if r1.QualiLapTime != 100357 {
		t.Errorf("r1.QualiLapTime: got %d want 100357", r1.QualiLapTime)
	}
	if r1.Laps != 7 {
		t.Errorf("r1.Laps: got %d want 7", r1.Laps)
	}
	// FinPos should be zero for quali
	if r1.FinPos != 0 {
		t.Errorf("r1.FinPos: got %d want 0", r1.FinPos)
	}

	r2 := parsed.Results[1]
	if r2.StartPos != 2 {
		t.Errorf("r2.StartPos: got %d want 2", r2.StartPos)
	}
	// 101.8151s → 101815ms (rounded)
	if r2.QualiLapTime != 101815 {
		t.Errorf("r2.QualiLapTime: got %d want 101815", r2.QualiLapTime)
	}
}

func TestParseXMLStringPayload(t *testing.T) {
	t.Parallel()

	parsed, err := ParseXML(minimalQualifyXML)
	if err != nil {
		t.Fatalf("ParseXML(string) returned unexpected error: %v", err)
	}

	if parsed.DataType != processor.ImportDataQuali {
		t.Fatalf("unexpected DataType: got %q", parsed.DataType)
	}
}

func TestParseXMLRFactorXMLPayload(t *testing.T) {
	t.Parallel()

	// Pass a pre-parsed *RFactorXML directly
	rfxl, err := unmarshalRFactorXML([]byte(minimalRaceXML))
	if err != nil {
		t.Fatalf("unmarshalRFactorXML: %v", err)
	}

	parsed, err := ParseXML(rfxl)
	if err != nil {
		t.Fatalf("ParseXML(*RFactorXML) returned unexpected error: %v", err)
	}

	if parsed.DataType != processor.ImportDataRace {
		t.Fatalf("unexpected DataType: got %q", parsed.DataType)
	}

	parsed2, err := ParseXML(*rfxl)
	if err != nil {
		t.Fatalf("ParseXML(RFactorXML) returned unexpected error: %v", err)
	}

	if parsed2.DataType != processor.ImportDataRace {
		t.Fatalf("unexpected DataType for value: got %q", parsed2.DataType)
	}
}

func TestParseXMLNilPointer(t *testing.T) {
	t.Parallel()

	var p *RFactorXML
	_, err := ParseXML(p)
	if err == nil {
		t.Fatal("expected error for nil *RFactorXML, got nil")
	}
}

func TestParseXMLEmptyPayload(t *testing.T) {
	t.Parallel()

	_, err := ParseXML([]byte(""))
	if err == nil {
		t.Fatal("expected error for empty payload, got nil")
	}
}

func TestParseXMLUnsupportedType(t *testing.T) {
	t.Parallel()

	_, err := ParseXML(42)
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}

func TestParseXMLNoSession(t *testing.T) {
	t.Parallel()

	noSessionXML := `<?xml version="1.0" encoding="utf-8"?>
<rFactorXML version="1.0">
<RaceResults>
<Setting>Multiplayer</Setting>
<TrackVenue>Test Track</TrackVenue>
</RaceResults>
</rFactorXML>`

	_, err := ParseXML([]byte(noSessionXML))
	if err == nil {
		t.Fatal("expected error when neither Race nor Qualify is present, got nil")
	}
}

func TestParseXMLRaceWithRealFile(t *testing.T) {
	t.Parallel()

	xmlData := loadTestFileOrSkip(t, "../../../../tmp/2026_03_13_22_42_51-53R1.xml")

	parsed, err := ParseXML(xmlData)
	if err != nil {
		t.Fatalf("ParseXML(race file) returned unexpected error: %v", err)
	}

	if parsed.DataType != processor.ImportDataRace {
		t.Fatalf("unexpected DataType: got %q want %q", parsed.DataType, processor.ImportDataRace)
	}

	if len(parsed.Results) == 0 {
		t.Fatal("expected non-empty results for real race file")
	}

	if parsed.Session.Track == "" {
		t.Fatal("expected non-empty Track")
	}
}

func TestParseXMLQualifyWithRealFile(t *testing.T) {
	t.Parallel()

	xmlData := loadTestFileOrSkip(t, "../../../../tmp/2026_03_13_20_33_19-49Q1.xml")

	parsed, err := ParseXML(xmlData)
	if err != nil {
		t.Fatalf("ParseXML(qualify file) returned unexpected error: %v", err)
	}

	if parsed.DataType != processor.ImportDataQuali {
		t.Fatalf("unexpected DataType: got %q want %q", parsed.DataType, processor.ImportDataQuali)
	}

	if len(parsed.Results) == 0 {
		t.Fatal("expected non-empty results for real qualify file")
	}
}

func loadTestFileOrSkip(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("skipping real-file test: %v", err)
	}

	return data
}
