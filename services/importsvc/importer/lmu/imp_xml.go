package lmu

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"time"

	processor "github.com/srlmgr/backend/services/importsvc/importer"
)

// ParseXML parses an LMU rFactor XML result payload.
// The payload may be a []byte, string, RFactorXML, or *RFactorXML.
// It detects whether the file is a race or qualifying session based on
// which sub-element (<Race> or <Qualify>) is present.
func ParseXML(payload any) (*processor.ParsedImportPayload, error) {
	data, err := payloadToRFactorXML(payload)
	if err != nil {
		return nil, err
	}

	rr := &data.RaceResults

	if rr.Race.DateTime != 0 {
		return parseRaceSession(rr)
	}

	if rr.Qualify.DateTime != 0 {
		return parseQualifySession(rr)
	}

	return nil, fmt.Errorf("xml payload contains neither a Race nor a Qualify session")
}

func payloadToRFactorXML(payload any) (*RFactorXML, error) {
	switch p := payload.(type) {
	case RFactorXML:
		return &p, nil
	case *RFactorXML:
		if p == nil {
			return nil, fmt.Errorf("invalid xml payload: nil RFactorXML")
		}

		return p, nil
	case []byte:
		return unmarshalRFactorXML(p)
	case string:
		return unmarshalRFactorXML([]byte(p))
	default:
		return nil, fmt.Errorf("unsupported xml payload type: %T", payload)
	}
}

func unmarshalRFactorXML(data []byte) (*RFactorXML, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("decode xml payload: empty payload")
	}

	var result RFactorXML
	if err := xml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode xml payload: %w", err)
	}

	return &result, nil
}

func parseRaceSession(rr *RaceResults) (*processor.ParsedImportPayload, error) {
	penaltyCountByName := mapPenaltyCountByDriverName(rr.Race.Stream.Penalties)
	results := make([]*processor.ResultRow, 0, len(rr.Race.Drivers))

	for i := range rr.Race.Drivers {
		d := rr.Race.Drivers[i]
		results = append(results, &processor.ResultRow{
			FinPos:         d.Position,
			CarID:          d.CarType,
			Car:            d.CarType,
			TeamID:         d.TeamName,
			DriverID:       d.Name,
			Name:           d.Name,
			CarNumber:      strconv.Itoa(d.CarNumber),
			FastestLapTime: secsToMS(d.BestLapTime),
			TotalTime:      secsToMS(d.FinishTime),
			Laps:           d.TotalLaps,
			Incidents:      penaltyCountByName[d.Name],
		})
	}

	return &processor.ParsedImportPayload{
		Session: processor.SessionInfo{
			StartTime: time.Unix(rr.Race.DateTime, 0).UTC(),
			Track:     rr.TrackVenue,
		},
		Results:  results,
		DataType: processor.ImportDataRace,
	}, nil
}

func parseQualifySession(rr *RaceResults) (*processor.ParsedImportPayload, error) {
	results := make([]*processor.ResultRow, 0, len(rr.Qualify.Drivers))

	for i := range rr.Qualify.Drivers {
		d := rr.Qualify.Drivers[i]
		results = append(results, &processor.ResultRow{
			StartPos:     d.Position,
			CarID:        d.CarType,
			Car:          d.CarType,
			TeamID:       d.TeamName,
			DriverID:     d.Name,
			Name:         d.Name,
			CarNumber:    strconv.Itoa(d.CarNumber),
			QualiLapTime: secsToMS(d.BestLapTime),
			Laps:         d.TotalLaps,
		})
	}

	return &processor.ParsedImportPayload{
		Session: processor.SessionInfo{
			StartTime: time.Unix(rr.Qualify.DateTime, 0).UTC(),
			Track:     rr.TrackVenue,
		},
		Results:  results,
		DataType: processor.ImportDataQuali,
	}, nil
}

// mapPenaltyCountByDriverName counts issued penalties (those with a non-empty Driver
// attribute) per driver name. Served-penalty notifications have no Driver attribute
// and are ignored.
func mapPenaltyCountByDriverName(penalties []Penalty) map[string]int {
	counts := make(map[string]int)
	for i := range penalties {
		if penalties[i].Driver != "" {
			counts[penalties[i].Driver]++
		}
	}

	return counts
}

// secsToMS converts a time in seconds (float64) to milliseconds (int).
func secsToMS(secs float64) int {
	return int(math.Round(secs * 1000))
}
