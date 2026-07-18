//nolint:tagliatelle // external definitions
package lmu

// RFactorXML is the root element of the LeMans Ultimate race result XML file.
type RFactorXML struct {
	Version     string      `xml:"version,attr"`
	RaceResults RaceResults `xml:"RaceResults"`
}

// RaceResults contains the complete race session configuration and results.
type RaceResults struct {
	Setting           string         `xml:"Setting"`
	ServerName        string         `xml:"ServerName"`
	ClientFuelVisible int            `xml:"ClientFuelVisible"`
	PlayerFile        string         `xml:"PlayerFile"`
	DateTime          int64          `xml:"DateTime"`
	TimeString        string         `xml:"TimeString"`
	TrackVenue        string         `xml:"TrackVenue"`
	TrackCourse       string         `xml:"TrackCourse"`
	TrackEvent        string         `xml:"TrackEvent"`
	TrackData         string         `xml:"TrackData"`
	TrackLength       float64        `xml:"TrackLength"`
	GameVersion       string         `xml:"GameVersion"`
	Dedicated         int            `xml:"Dedicated"`
	ConnectionType    ConnectionType `xml:"ConnectionType"`
	RaceLaps          int            `xml:"RaceLaps"`
	RaceTime          int            `xml:"RaceTime"`
	MechFailRate      int            `xml:"MechFailRate"`
	DamageMult        int            `xml:"DamageMult"`
	FuelMult          int            `xml:"FuelMult"`
	TireMult          int            `xml:"TireMult"`
	VehiclesAllowed   string         `xml:"VehiclesAllowed"`
	ParcFerme         int            `xml:"ParcFerme"`
	FixedSetups       int            `xml:"FixedSetups"`
	FreeSettings      int            `xml:"FreeSettings"`
	FixedUpgrades     int            `xml:"FixedUpgrades"`
	TireWarmers       int            `xml:"TireWarmers"`
	Race              Race           `xml:"Race"`
	Qualify           Race           `xml:"Qualify"`
}

// ConnectionType represents the server connection configuration.
type ConnectionType struct {
	Upload   int    `xml:"upload,attr"`
	Download int    `xml:"download,attr"`
	Value    string `xml:",chardata"`
}

// Race contains the race session data including the event stream
// and per-driver results.
type Race struct {
	DateTime   int64    `xml:"DateTime"`
	TimeString string   `xml:"TimeString"`
	Laps       int      `xml:"Laps"`
	Minutes    int      `xml:"Minutes"`
	Stream     Stream   `xml:"Stream"`
	Drivers    []Driver `xml:"Driver"`
}

// Stream contains the ordered sequence of events that occurred during the race.
type Stream struct {
	Penalties   []Penalty    `xml:"Penalty"`
	Incidents   []Incident   `xml:"Incident"`
	TrackLimits []TrackLimit `xml:"TrackLimits"`
	Sectors     []Sector     `xml:"Sector"`
	Scores      []Score      `xml:"Score"`
}

// Penalty represents either a penalty being issued to a driver or a notification
// that a penalty was served.
// When Driver is empty the entry represents a served penalty notification.
type Penalty struct {
	Driver      string  `xml:"Driver,attr"`
	ID          int     `xml:"ID,attr"`
	PenaltyType string  `xml:"Penalty,attr"`
	Time        int     `xml:"Time,attr"`
	Laps        int     `xml:"Laps,attr"`
	Reason      string  `xml:"Reason,attr"`
	Et          float64 `xml:"et,attr"`
	Text        string  `xml:",chardata"`
}

// Incident represents a contact/collision event reported by a driver.
type Incident struct {
	Et   float64 `xml:"et,attr"`
	Text string  `xml:",chardata"`
}

// TrackLimit represents a track limits violation or adjudication event.
type TrackLimit struct {
	Driver        string  `xml:"Driver,attr"`
	ID            int     `xml:"ID,attr"`
	Lap           int     `xml:"Lap,attr"`
	WarningPoints float64 `xml:"WarningPoints,attr"`
	CurrentPoints float64 `xml:"CurrentPoints,attr"`
	Resolution    int     `xml:"Resolution,attr"`
	Et            float64 `xml:"et,attr"`
	Text          string  `xml:",chardata"`
}

// Sector represents a new class-best sector time event.
type Sector struct {
	Driver string  `xml:"Driver,attr"`
	ID     int     `xml:"ID,attr"`
	Sector int     `xml:"Sector,attr"`
	Class  string  `xml:"Class,attr"`
	Et     float64 `xml:"et,attr"`
	Text   string  `xml:",chardata"`
}

// Score represents a lap scoring update for a driver.
type Score struct {
	Et   float64 `xml:"et,attr"`
	Text string  `xml:",chardata"`
}

// Driver contains the configuration, classification,
// and lap-by-lap data for a single driver.
type Driver struct {
	Name                   string         `xml:"Name"`
	Connected              int            `xml:"Connected"`
	VehFile                string         `xml:"VehFile"`
	UpgradeCode            string         `xml:"UpgradeCode"`
	VehName                string         `xml:"VehName"`
	Category               string         `xml:"Category"`
	CarType                string         `xml:"CarType"`
	CarClass               string         `xml:"CarClass"`
	CarNumber              int            `xml:"CarNumber"`
	TeamName               string         `xml:"TeamName"`
	IsPlayer               int            `xml:"isPlayer"`
	ServerScored           int            `xml:"ServerScored"`
	GridPos                int            `xml:"GridPos"`
	Position               int            `xml:"Position"`
	ClassGridPos           int            `xml:"ClassGridPos"`
	ClassPosition          int            `xml:"ClassPosition"`
	LapRankIncludingDiscos int            `xml:"LapRankIncludingDiscos"`
	LapData                []Lap          `xml:"Lap"`
	BestLapTime            float64        `xml:"BestLapTime"`
	FinishTime             float64        `xml:"FinishTime"`
	TotalLaps              int            `xml:"Laps"`
	Pitstops               int            `xml:"Pitstops"`
	FinishStatus           string         `xml:"FinishStatus"`
	ControlAndAids         ControlAndAids `xml:"ControlAndAids"`
}

// Lap contains all telemetry and timing data recorded for a single lap.
// LapTime holds the lap time as a string because invalid laps are
// represented as "--.----".
type Lap struct {
	Num         int     `xml:"num,attr"`
	Position    int     `xml:"p,attr"`
	ElapsedTime string  `xml:"et,attr"`
	Sector1     float64 `xml:"s1,attr"`
	Sector2     float64 `xml:"s2,attr"`
	Sector3     float64 `xml:"s3,attr"`
	TopSpeed    float64 `xml:"topspeed,attr"`
	Fuel        float64 `xml:"fuel,attr"`
	FuelUsed    float64 `xml:"fuelUsed,attr"`
	Ve          float64 `xml:"ve,attr"`
	VeUsed      float64 `xml:"veUsed,attr"`
	TireWearFL  float64 `xml:"twfl,attr"`
	TireWearFR  float64 `xml:"twfr,attr"`
	TireWearRL  float64 `xml:"twrl,attr"`
	TireWearRR  float64 `xml:"twrr,attr"`
	FCompound   string  `xml:"fcompound,attr"`
	RCompound   string  `xml:"rcompound,attr"`
	TireFL      string  `xml:"FL,attr"`
	TireFR      string  `xml:"FR,attr"`
	TireRL      string  `xml:"RL,attr"`
	TireRR      string  `xml:"RR,attr"`
	Pit         int     `xml:"pit,attr"`
	LapTime     string  `xml:",chardata"`
}

// ControlAndAids records which control aids were active for a driver
// across a lap range.
type ControlAndAids struct {
	StartLap int    `xml:"startLap,attr"`
	EndLap   int    `xml:"endLap,attr"`
	Value    string `xml:",chardata"`
}
