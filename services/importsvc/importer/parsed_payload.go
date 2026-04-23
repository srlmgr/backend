package importer

import (
	"time"
)

// ParsedImportPayload is the common parsed payload shape returned by CSV processors.
type ParsedImportPayload struct {
	Session  SessionInfo
	Results  []*ResultRow
	DataType ImportData
}

// SessionInfo contains event-level data from an import payload.
type SessionInfo struct {
	StartTime time.Time
	Track     string
}

// ResultRow contains normalized per-driver result values extracted from payloads.
// Note: IDs are specific to the simulation and must be resolved later
type (
	ResultRow struct {
		FinPos         int
		CarID          string
		Car            string
		TeamID         string
		DriverID       string
		Name           string
		StartPos       int
		CarNumber      string
		Interval       string //
		LapsLed        int
		QualiLapTime   int // in ms
		TotalTime      int // in ms
		FastestLapTime int // in ms
		Laps           int
		Incidents      int
		TeamDrivers    []*TeamDriver // filled in team-based events
	}
	TeamDriver struct {
		DriverID       string
		Name           string
		FastestLapTime int // in ms
		Laps           int
		Incidents      int
	}
)
