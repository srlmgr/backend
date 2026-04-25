package acc

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	processor "github.com/srlmgr/backend/services/importsvc/importer"
)

func ParseJSON(payload any) (*processor.ParsedImportPayload, error) {
	data, err := payloadToEventData(payload)
	if err != nil {
		return nil, err
	}

	results := make([]*processor.ResultRow, 0, len(data.SessionResult.LeaderBoardRows))
	penaltyCountByCarID := mapPenaltyCountByCarID(data.Penalties, data.PostRacePenalties)

	for i := range data.SessionResult.LeaderBoardRows {
		entry := data.SessionResult.LeaderBoardRows[i]
		result := &processor.ResultRow{
			CarID:    strconv.Itoa(entry.Car.CarModel),
			Car:      strconv.Itoa(entry.Car.CarModel),
			TeamID:   strconv.Itoa(entry.Car.TeamGUID),
			DriverID: entry.CurrentDriver.PlayerID,
			Name: strings.TrimSpace(
				entry.CurrentDriver.FirstName + " " + entry.CurrentDriver.LastName,
			),
			CarNumber: strconv.Itoa(entry.Car.RaceNumber),
		}

		if strings.EqualFold(data.SessionType, "q") {
			result.StartPos = i + 1
			result.QualiLapTime = entry.Timing.BestLap
		}
		if strings.EqualFold(data.SessionType, "r") {
			result.FinPos = i + 1
			result.FastestLapTime = entry.Timing.BestLap
			result.TotalTime = entry.Timing.TotalTime
			result.Laps = entry.Timing.LapCount
			result.Incidents = penaltyCountByCarID[entry.Car.CarID]
		}

		results = append(results, result)
	}

	return &processor.ParsedImportPayload{
		Session: processor.SessionInfo{
			StartTime: time.Time{},
			Track:     data.TrackName,
		},
		Results:  results,
		DataType: detectImportDataType(data.SessionType),
	}, nil
}

func detectImportDataType(sessionType string) processor.ImportData {
	if strings.EqualFold(sessionType, "q") {
		return processor.ImportDataQuali
	}
	if strings.EqualFold(sessionType, "r") {
		return processor.ImportDataRace
	}

	return processor.ImportDataAll
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
		return unmarshalEventData(p)
	case string:
		return unmarshalEventData([]byte(p))
	default:
		return nil, fmt.Errorf("unsupported json payload type: %T", payload)
	}
}

func unmarshalEventData(payload []byte) (*EventData, error) {
	normalized, err := normalizeJSONBytes(payload)
	if err != nil {
		return nil, err
	}

	var data EventData
	if err := json.Unmarshal(normalized, &data); err != nil {
		return nil, fmt.Errorf("decode json payload: %w", err)
	}

	return &data, nil
}

func normalizeJSONBytes(raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("decode json payload: empty payload")
	}

	if json.Valid(trimmed) {
		return trimmed, nil
	}

	decoded, err := decodeUTF16(trimmed)
	if err != nil {
		return nil, fmt.Errorf("decode json payload: %w", err)
	}

	decoded = bytes.TrimSpace(decoded)
	if !json.Valid(decoded) {
		return nil, fmt.Errorf("decode json payload: invalid json content")
	}

	return decoded, nil
}

func decodeUTF16(raw []byte) ([]byte, error) {
	if len(raw)%2 != 0 {
		return nil, fmt.Errorf("invalid utf-16 payload length")
	}

	var order binary.ByteOrder = binary.LittleEndian
	start := 0
	if len(raw) >= 2 {
		switch {
		case raw[0] == 0xFF && raw[1] == 0xFE:
			order = binary.LittleEndian
			start = 2
		case raw[0] == 0xFE && raw[1] == 0xFF:
			order = binary.BigEndian
			start = 2
		default:
			order = detectUTF16ByteOrder(raw)
		}
	}

	if (len(raw)-start)%2 != 0 {
		return nil, fmt.Errorf("invalid utf-16 payload length")
	}

	u16 := make([]uint16, (len(raw)-start)/2)
	for i := range u16 {
		offset := start + i*2
		u16[i] = order.Uint16(raw[offset : offset+2])
	}

	runes := utf16.Decode(u16)
	return []byte(string(runes)), nil
}

func detectUTF16ByteOrder(raw []byte) binary.ByteOrder {
	var zeroEven int
	var zeroOdd int
	for i := 0; i+1 < len(raw); i += 2 {
		if raw[i] == 0 {
			zeroEven++
		}
		if raw[i+1] == 0 {
			zeroOdd++
		}
	}

	if zeroOdd >= zeroEven {
		return binary.LittleEndian
	}

	return binary.BigEndian
}

func mapPenaltyCountByCarID(groups ...[]Penalty) map[int]int {
	byCarID := make(map[int]int)
	for i := range groups {
		for j := range groups[i] {
			penalty := groups[i][j]
			byCarID[penalty.CarID]++
		}
	}

	return byCarID
}
