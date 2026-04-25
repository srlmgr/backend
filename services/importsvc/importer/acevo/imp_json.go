package acevo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	processor "github.com/srlmgr/backend/services/importsvc/importer"
)

//nolint:funlen,gocritic // by design
func ParseJSON(payload any) (*processor.ParsedImportPayload, error) {
	session, err := payloadToSession(payload)
	if err != nil {
		return nil, err
	}

	driverByUUID := buildDriverByUUID(session.Drivers)
	carByUUID := buildCarByUUID(session.Cars)
	driverToCarUUID := buildDriverToCarUUID(session.Laps)
	driverLapStats := buildDriverLapStats(session.Laps)

	results := make([]*processor.ResultRow, 0, len(session.DriverStandings))

	for i, driverGUID := range session.DriverStandings {
		uuid := driverGUID.UUID()
		driver, ok := driverByUUID[uuid]
		if !ok {
			continue
		}

		result := &processor.ResultRow{
			DriverID: driver.PlayerID,
			Name:     strings.TrimSpace(driver.FirstName + " " + driver.LastName),
		}

		if carUUID, ok := driverToCarUUID[uuid]; ok {
			if car, ok := carByUUID[carUUID]; ok {
				result.CarID = car.ModelDisplayName
				result.Car = car.ModelDisplayName
				result.CarNumber = strconv.Itoa(car.RaceNumber)
			}
		}

		stats := driverLapStats[uuid]

		if strings.EqualFold(session.SessionType, "qualify") {
			result.StartPos = i + 1
			if i < len(session.TimeStandings) {
				result.QualiLapTime = session.TimeStandings[i]
			}
		} else if strings.EqualFold(session.SessionType, "race") {
			result.FinPos = i + 1
			if i < len(session.TimeStandings) {
				result.TotalTime = session.TimeStandings[i]
			}
			result.Laps = stats.lapCount
			result.FastestLapTime = stats.fastestLap
		}

		results = append(results, result)
	}

	return &processor.ParsedImportPayload{
		Session: processor.SessionInfo{
			StartTime: time.Time{},
			Track:     session.TrackName,
		},
		Results:  results,
		DataType: detectDataType(session.SessionType),
	}, nil
}

func detectDataType(sessionType string) processor.ImportData {
	if strings.EqualFold(sessionType, "qualify") {
		return processor.ImportDataQuali
	}
	if strings.EqualFold(sessionType, "race") {
		return processor.ImportDataRace
	}
	return processor.ImportDataAll
}

func payloadToSession(payload any) (*Session, error) {
	switch p := payload.(type) {
	case Session:
		return &p, nil
	case *Session:
		if p == nil {
			return nil, fmt.Errorf("invalid json payload: nil Session")
		}
		return p, nil
	case []byte:
		return unmarshalSession(p)
	case string:
		return unmarshalSession([]byte(p))
	default:
		return nil, fmt.Errorf("unsupported json payload type: %T", payload)
	}
}

func unmarshalSession(data []byte) (*Session, error) {
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("decode json payload: %w", err)
	}
	return &s, nil
}

// lapStats holds per-driver aggregated lap metrics.
type lapStats struct {
	lapCount   int
	fastestLap int
}

// buildDriverByUUID returns a map keyed by the full UUID of each driver's GUID.
func buildDriverByUUID(drivers []Driver) map[string]Driver {
	m := make(map[string]Driver, len(drivers))
	for i := range drivers {
		m[drivers[i].GUID.UUID()] = drivers[i]
	}
	return m
}

// buildCarByUUID returns a map keyed by the full UUID of each car's car_id GUID.
func buildCarByUUID(cars []Car) map[string]Car {
	m := make(map[string]Car, len(cars))
	for i := range cars {
		m[cars[i].CarID.UUID()] = cars[i]
	}
	return m
}

// buildDriverToCarUUID returns a map from driver UUID → car UUID,
// derived from the laps list (first occurrence wins).
func buildDriverToCarUUID(laps []Lap) map[string]string {
	m := make(map[string]string)
	for i := range laps {
		driverUUID := laps[i].DriverKey.UUID()
		if _, exists := m[driverUUID]; !exists {
			m[driverUUID] = laps[i].CarKey.UUID()
		}
	}
	return m
}

// buildDriverLapStats returns per-driver lap count and fastest valid lap time.
// Laps with flags == 2 are treated as clean laps eligible for best-lap tracking.
func buildDriverLapStats(laps []Lap) map[string]lapStats {
	m := make(map[string]lapStats)
	for i := range laps {
		uuid := laps[i].DriverKey.UUID()
		st := m[uuid]
		st.lapCount++
		if laps[i].Flags == 2 && laps[i].Time > 0 {
			if st.fastestLap == 0 || laps[i].Time < st.fastestLap {
				st.fastestLap = laps[i].Time
			}
		}
		m[uuid] = st
	}
	return m
}
