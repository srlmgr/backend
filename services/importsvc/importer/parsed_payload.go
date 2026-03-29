package importer

import (
	"time"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
)

// ParsedImportPayload is the common parsed payload shape returned by CSV processors.
type ParsedImportPayload struct {
	Session SessionInfo
	Results []ResultRow
}

// SessionInfo contains event-level data from an import payload.
type SessionInfo struct {
	StartTime time.Time
	Track     string
}

// ResultRow contains normalized per-driver result values extracted from payloads.
// Note: IDs are specific to the simulation and must be resolved later
type ResultRow struct {
	FinPos         int
	CarID          string
	Car            string
	TeamID         string
	DriverID       string
	Name           string
	StartPos       int
	CarNumber      string
	Interval       string
	LapsLed        int
	FastestLapTime string
	Laps           int
	Incidents      int
}
type ParsedImportProcessor struct{}

func NewParsedImportProcessor() *ParsedImportProcessor {
	return &ParsedImportProcessor{}
}

func (p *ParsedImportProcessor) Process() []*commonv1.ResultEntry {
	return nil
}
