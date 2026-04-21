//nolint:tagliatelle // external definition
package acc

//nolint:tagliatelle // external definition
type EventData struct {
	SessionType       string    `json:"sessionType"`
	TrackName         string    `json:"trackName"`
	SessionIndex      int       `json:"sessionIndex"`
	RaceWeekendIndex  int       `json:"raceWeekendIndex"`
	MetaData          string    `json:"metaData"`
	ServerName        string    `json:"serverName"`
	SessionResult     Session   `json:"sessionResult"`
	Laps              []Lap     `json:"laps"`
	Penalties         []Penalty `json:"penalties"`
	PostRacePenalties []Penalty `json:"post_race_penalties"`
}

type Session struct {
	BestLap         int                `json:"bestlap"`
	BestSplits      []int              `json:"bestSplits"`
	IsWetSession    int                `json:"isWetSession"`
	Type            int                `json:"type"`
	LeaderBoardRows []LeaderBoardEntry `json:"leaderBoardLines"`
}

type LeaderBoardEntry struct {
	Car                     Car             `json:"car"`
	CurrentDriver           Driver          `json:"currentDriver"`
	CurrentDriverIndex      int             `json:"currentDriverIndex"`
	DriverTotalTimes        []float64       `json:"driverTotalTimes"`
	MissingMandatoryPitstop int             `json:"missingMandatoryPitstop"`
	Timing                  LeaderBoardTime `json:"timing"`
}

type Car struct {
	CarID       int      `json:"carId"`
	RaceNumber  int      `json:"raceNumber"`
	CarModel    int      `json:"carModel"`
	CupCategory int      `json:"cupCategory"`
	CarGroup    string   `json:"carGroup"`
	TeamName    string   `json:"teamName"`
	Nationality int      `json:"nationality"`
	CarGUID     int      `json:"carGuid"`
	TeamGUID    int      `json:"teamGuid"`
	Drivers     []Driver `json:"drivers"`
}

type Driver struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	ShortName string `json:"shortName"`
	PlayerID  string `json:"playerId"`
}

type LeaderBoardTime struct {
	BestLap     int   `json:"bestLap"`
	BestSplits  []int `json:"bestSplits"`
	LapCount    int   `json:"lapCount"`
	LastLap     int   `json:"lastLap"`
	LastSplitID int   `json:"lastSplitId"`
	LastSplits  []int `json:"lastSplits"`
	TotalTime   int   `json:"totalTime"`
}

type Lap struct {
	CarID          int   `json:"carId"`
	DriverIndex    int   `json:"driverIndex"`
	LapTime        int   `json:"laptime"`
	IsValidForBest bool  `json:"isValidForBest"`
	Splits         []int `json:"splits"`
}

type Penalty struct {
	CarID          int    `json:"carId"`
	DriverIndex    int    `json:"driverIndex"`
	Reason         string `json:"reason"`
	Penalty        string `json:"penalty"`
	PenaltyValue   int    `json:"penaltyValue"`
	ViolationInLap int    `json:"violationInLap"`
	ClearedInLap   int    `json:"clearedInLap"`
}
