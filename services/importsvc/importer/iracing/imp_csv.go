package iracing

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	processor "github.com/srlmgr/backend/services/importsvc/importer"
)

// ParseCSV parses iRacing CSV payloads where two CSV blocks are
// separated by an empty line.
func ParseCSV(payload any) (*processor.ParsedImportPayload, error) {
	payloadText, err := payloadToString(payload)
	if err != nil {
		return nil, err
	}

	records, err := readCSVRecords(payloadText)
	if err != nil {
		return nil, err
	}
	if len(records) < 4 {
		return nil, fmt.Errorf("invalid csv payload: expected two csv sections")
	}

	secondHeaderIndex, err := findSecondCSVHeaderIndex(records)
	if err != nil {
		return nil, err
	}
	if secondHeaderIndex < 2 || secondHeaderIndex >= len(records)-1 {
		return nil, fmt.Errorf("invalid csv payload: second csv section has no data rows")
	}

	session, err := parseSessionCSV(records[0], records[1])
	if err != nil {
		return nil, err
	}

	results, err := parseResultsCSV(
		records[secondHeaderIndex],
		records[secondHeaderIndex+1:])
	if err != nil {
		return nil, err
	}

	return &processor.ParsedImportPayload{
		Session: session,
		Results: results,
	}, nil
}

func payloadToString(payload any) (string, error) {
	switch p := payload.(type) {
	case string:
		return p, nil
	case []byte:
		return string(p), nil
	default:
		return "", fmt.Errorf("unsupported csv payload type: %T", payload)
	}
}

func findSecondCSVHeaderIndex(records [][]string) (int, error) {
	for i := 2; i < len(records); i++ {
		if hasHeaderFields(records[i], "Fin Pos", "Car ID") {
			return i, nil
		}
	}

	return -1, fmt.Errorf("invalid csv payload: second csv header not found")
}

func hasHeaderFields(header []string, fields ...string) bool {
	seen := make(map[string]struct{}, len(header))
	for _, value := range header {
		seen[strings.TrimSpace(value)] = struct{}{}
	}

	for _, field := range fields {
		if _, ok := seen[field]; !ok {
			return false
		}
	}

	return true
}

func parseSessionCSV(header, record []string) (processor.SessionInfo, error) {
	row := rowMap(header, record)
	startTimeRaw, ok := row["Start Time"]
	if !ok || strings.TrimSpace(startTimeRaw) == "" {
		return processor.SessionInfo{}, fmt.Errorf(
			"missing required field in first csv section: Start Time",
		)
	}

	startTime, err := time.Parse(time.RFC3339, startTimeRaw)
	if err != nil {
		return processor.SessionInfo{}, fmt.Errorf("parse Start Time: %w", err)
	}

	track, ok := row["Track"]
	if !ok || strings.TrimSpace(track) == "" {
		return processor.SessionInfo{}, fmt.Errorf(
			"missing required field in first csv section: Track",
		)
	}

	return processor.SessionInfo{
		StartTime: startTime,
		Track:     track,
	}, nil
}

//nolint:funlen,whitespace,gocyclo // many attributes to parse
func parseResultsCSV(header []string, records [][]string) (
	[]processor.ResultRow, error,
) {
	rows := make([]processor.ResultRow, 0, len(records))

	for i, record := range records {
		mapped := rowMap(header, record)

		finPos, err := requiredIntField(mapped, "Fin Pos")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		carID, err := requiredStringField(mapped, "Car ID")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		car, err := requiredStringField(mapped, "Car")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		teamID, err := requiredStringField(mapped, "Team ID")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		custID, err := requiredStringField(mapped, "Cust ID")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		name, err := requiredStringField(mapped, "Name")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		startPos, err := requiredIntField(mapped, "Start Pos")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		carNumber, err := requiredStringField(mapped, "Car #")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		interval, err := requiredStringField(mapped, "Interval")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		lapsLed, err := requiredIntField(mapped, "Laps Led")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		qualifyLapTime, err := laptimeField(mapped, "Qualify Time")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		avgLapTime, err := laptimeField(mapped, "Average Lap Time")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		fastestLapTime, err := laptimeField(mapped, "Fastest Lap Time")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		laps, err := requiredIntField(mapped, "Laps Comp")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		inc, err := requiredIntField(mapped, "Inc")
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}

		rows = append(rows, processor.ResultRow{
			FinPos:         finPos,
			CarID:          carID,
			Car:            car,
			TeamID:         teamID,
			DriverID:       custID,
			Name:           name,
			StartPos:       startPos,
			CarNumber:      carNumber,
			Interval:       interval,
			LapsLed:        lapsLed,
			QualiLapTime:   qualifyLapTime,
			TotalTime:      avgLapTime * laps,
			FastestLapTime: fastestLapTime,
			Laps:           laps,
			Incidents:      inc,
		})
	}

	return rows, nil
}

func readCSVRecords(payload string) ([][]string, error) {
	normalized := strings.ReplaceAll(payload, "\r\n", "\n")
	reader := csv.NewReader(strings.NewReader(normalized))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	return records, nil
}

func rowMap(header, record []string) map[string]string {
	row := make(map[string]string, len(header))
	for i, key := range header {
		if i >= len(record) {
			row[key] = ""
			continue
		}
		row[key] = record[i]
	}

	return row
}

func requiredStringField(row map[string]string, field string) (string, error) {
	raw, ok := row[field]
	if !ok {
		return "", fmt.Errorf("missing required field: %s", field)
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("empty required field: %s", field)
	}

	return value, nil
}

//nolint:funlen // complex parsing logic
func laptimeField(row map[string]string, field string) (int, error) {
	raw, ok := row[field]
	if !ok {
		return -1, nil
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return -1, nil
	}

	durationRe := regexp.MustCompile(
		`^(?:(?P<minutes>\d{1,2}):)?(?P<seconds>\d{1,2})\.(?P<milliseconds>\d{3})$`,
	)

	match := durationRe.FindStringSubmatch(value)
	if match == nil {
		return -1, fmt.Errorf("invalid duration: %s", value)
	}

	// Map group names → values
	result := make(map[string]string)
	for i, name := range durationRe.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	var minutes int
	var seconds int
	var millis int
	var err error

	if result["minutes"] != "" {
		minutes, err = strconv.Atoi(result["minutes"])
		if err != nil {
			return -1, err
		}
	}

	seconds, err = strconv.Atoi(result["seconds"])
	if err != nil {
		return -1, err
	}

	millis, err = strconv.Atoi(result["milliseconds"])
	if err != nil {
		return -1, err
	}

	total := minutes*60_000 + seconds*1_000 + millis
	return total, nil
}

func requiredIntField(row map[string]string, field string) (int, error) {
	value, err := requiredStringField(row, field)
	if err != nil {
		return 0, err
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid int for field %s: %w", field, err)
	}

	return parsed, nil
}
