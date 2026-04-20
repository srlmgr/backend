//nolint:tagliatelle // external definition
package r3e

// EventData is the top-level payload exported by the RaceRoom event endpoint.
type EventData struct {
	Server            string    `json:"Server"`
	StartTime         int64     `json:"StartTime"`
	Time              int64     `json:"Time"`
	Experience        string    `json:"Experience"`
	Difficulty        string    `json:"Difficulty"`
	FuelUsage         string    `json:"FuelUsage"`
	TireWear          string    `json:"TireWear"`
	MechanicalDamage  *string   `json:"MechanicalDamage"`
	FlagRules         string    `json:"FlagRules"`
	CutRules          string    `json:"CutRules"`
	RaceSeriesFormat  *string   `json:"RaceSeriesFormat"`
	WreckerPrevention string    `json:"WreckerPrevention"`
	MandatoryPitstop  string    `json:"MandatoryPitstop"`
	Track             string    `json:"Track"`
	TrackLayout       string    `json:"TrackLayout"`
	Sessions          []Session `json:"Sessions"`
}

type Session struct {
	Type    string   `json:"Type"`
	Players []Player `json:"Players"`
}

type Player struct {
	UserID               int    `json:"UserId"`
	FullName             string `json:"FullName"`
	Username             string `json:"Username"`
	UserWeightPenalty    int    `json:"UserWeightPenalty"`
	CarID                int    `json:"CarId"`
	Car                  string `json:"Car"`
	CarWeightPenalty     int    `json:"CarWeightPenalty"`
	LiveryID             int    `json:"LiveryId"`
	CarPerformanceIndex  int    `json:"CarPerformanceIndex"`
	Position             int    `json:"Position"`
	PositionInClass      int    `json:"PositionInClass"`
	StartPosition        int    `json:"StartPosition"`
	StartPositionInClass int    `json:"StartPositionInClass"`
	BestLapTime          int    `json:"BestLapTime"`
	TotalTime            int    `json:"TotalTime"`
	FinishStatus         string `json:"FinishStatus"`
	RaceSessionLaps      []Lap  `json:"RaceSessionLaps"`
}

type Lap struct {
	Time            int        `json:"Time"`
	SectorTimes     []int      `json:"SectorTimes"`
	PositionInClass int        `json:"PositionInClass"`
	Valid           bool       `json:"Valid"`
	Position        int        `json:"Position"`
	PitStopOccured  bool       `json:"PitStopOccured"`
	Incidents       []Incident `json:"Incidents"`
}

type Incident struct {
	Type        int `json:"Type"`
	Points      int `json:"Points"`
	OtherUserID int `json:"OtherUserId"`
}
