package r3e

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	processor "github.com/srlmgr/backend/services/importsvc/importer"
)

func ParseJSON(payload any) (*processor.ParsedImportPayload, error) {
	data, err := payloadToEventData(payload)
	if err != nil {
		return nil, err
	}

	raceSession, ok := findSession(data.Sessions, "race")
	if !ok {
		return nil, fmt.Errorf("missing required session type: race")
	}

	qualiBestLapByUserID := mapQualifyingBestLaps(data.Sessions)
	results := make([]*processor.ResultRow, 0, len(raceSession.Players))
	for i := range raceSession.Players {
		player := raceSession.Players[i]
		row := &processor.ResultRow{
			FinPos:         player.Position,
			CarID:          strconv.Itoa(player.CarID),
			Car:            player.Car,
			TeamID:         strconv.Itoa(player.UserID),
			DriverID:       strconv.Itoa(player.UserID),
			Name:           player.FullName,
			StartPos:       player.StartPosition,
			TotalTime:      player.TotalTime,
			FastestLapTime: player.BestLapTime,
			Laps:           len(player.RaceSessionLaps),
			Incidents:      sumIncidentPoints(player.RaceSessionLaps),
		}

		if qualiBestLap, ok := qualiBestLapByUserID[player.UserID]; ok {
			row.QualiLapTime = qualiBestLap
		}

		results = append(results, row)
	}

	return &processor.ParsedImportPayload{
		Session: processor.SessionInfo{
			StartTime: time.Unix(data.StartTime, 0).UTC(),
			Track:     data.Track + " - " + data.TrackLayout,
		},
		Results: results,
	}, nil
}

func payloadToEventData(payload any) (*EventData, error) {
	switch p := payload.(type) {
	case EventData:
		return &p, nil
	case *EventData:
		if p == nil {
			return nil, fmt.Errorf("invalid json payload: nil EventData")
		}

		return p, nil
	case []byte:
		var data EventData
		if err := json.Unmarshal(p, &data); err != nil {
			return nil, fmt.Errorf("decode json payload: %w", err)
		}

		return &data, nil
	case string:
		var data EventData
		if err := json.Unmarshal([]byte(p), &data); err != nil {
			return nil, fmt.Errorf("decode json payload: %w", err)
		}

		return &data, nil
	default:
		return nil, fmt.Errorf("unsupported json payload type: %T", payload)
	}
}

func findSession(sessions []Session, sessionType string) (Session, bool) {
	for i := range sessions {
		session := sessions[i]
		if strings.EqualFold(session.Type, sessionType) {
			return session, true
		}
	}

	return Session{}, false
}

func mapQualifyingBestLaps(sessions []Session) map[int]int {
	qualifying, ok := findSession(sessions, "qualify")
	if !ok {
		return nil
	}

	bestByUserID := make(map[int]int, len(qualifying.Players))
	for i := range qualifying.Players {
		player := qualifying.Players[i]
		bestByUserID[player.UserID] = player.BestLapTime
	}

	return bestByUserID
}

func sumIncidentPoints(laps []Lap) int {
	total := 0
	for i := range laps {
		lap := laps[i]
		for _, inc := range lap.Incidents {
			total += inc.Points
		}
	}

	return total
}
