package iracing

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	processor "github.com/srlmgr/backend/grpc/services/importsvc/importer"
	"github.com/srlmgr/backend/support/iracing/irdata"
)

// ParseJSON parses a single-race iRacing JSON payload, returning the first
// detected race (or the qualifying-only session, if no race is present).
//
//nolint:whitespace // editor/linter issue
func ParseJSON(
	ctx context.Context,
	payload any,
) (*processor.ParsedImportPayload, error) {
	payloads, err := ParseJSONMultiRace(ctx, payload)
	if err != nil {
		return nil, err
	}

	return payloads[0], nil
}

// ParseJSONMultiRace parses an iRacing JSON payload that may contain multiple races
// (heats). It returns one ParsedImportPayload per detected race, in the order the race
// sessions appear in session_results, with RaceSequenceNo set (1-based) when more than
// one race is present. If no race session exists, a single qualifying-only payload with
// RaceSequenceNo left at zero is returned.
//
//nolint:whitespace // editor/linter issue
func ParseJSONMultiRace(
	ctx context.Context, payload any,
) ([]*processor.ParsedImportPayload, error) {
	p, err := newJSONProcessor(ctx, payload)
	if err != nil {
		return nil, err
	}

	return p.process(), nil
}

// jsonProcessor holds the state derived from an iRacing event result that is shared
// by all steps of building the parsed payloads.
type jsonProcessor struct {
	ctx                     context.Context
	data                    *EventResult
	isTeamRace              bool
	raceSessions            []SimSession
	qualiSession            SimSession
	hasQuali                bool
	qualiBestLapByResultKey map[string]int
	teamDriverIDs           TeamDriverIDs
	lapIncidents            incsBySession
}

func newJSONProcessor(ctx context.Context, payload any) (*jsonProcessor, error) {
	envelope, err := payloadToEventResult(payload)
	if err != nil {
		return nil, err
	}

	data := &envelope.Data
	p := &jsonProcessor{
		ctx:          ctx,
		data:         data,
		isTeamRace:   data.MaxTeamDrivers > 1,
		raceSessions: findRaceSessions(data.SessionResults),
		lapIncidents: make(incsBySession),
	}
	p.qualiSession, p.hasQuali = findFirstQualifySession(data.SessionResults)
	if len(p.raceSessions) == 0 && !p.hasQuali {
		return nil, fmt.Errorf("json payload contains neither race nor qualifying sessions")
	}

	p.qualiBestLapByResultKey = buildQualiBestLapMap(&p.qualiSession, p.hasQuali)
	p.teamDriverIDs = collectTeamDriverIDs(data.SessionResults)

	return p, nil
}

// process builds one payload per race session, or a single qualifying-only payload
// when no race session exists.
func (p *jsonProcessor) process() []*processor.ParsedImportPayload {
	if len(p.raceSessions) == 0 {
		return []*processor.ParsedImportPayload{p.buildRacePayload(nil, true)}
	}
	p.collectIncidents()

	payloads := make([]*processor.ParsedImportPayload, len(p.raceSessions))
	for i := range p.raceSessions {
		// Qualifying only sets the grid for the first race of a heat; later heats/the
		// feature race start from their own standings, so quali data must not carry over.
		built := p.buildRacePayload(&p.raceSessions[i], i == 0)
		if len(p.raceSessions) > 1 {
			built.RaceSequenceNo = i + 1
		}
		payloads[i] = built
	}

	return payloads
}

func (p *jsonProcessor) collectIncidents() {
	if api, ok := irdata.FromContext(p.ctx); ok {
		// Implementation for computing offtracks goes here.
		collector := newLapDataCollector(api)
		incs := collector.run(p.data)
		p.lapIncidents = incs
	}
}

// buildRacePayload builds a ParsedImportPayload for a single race session (or, when
// raceSession is nil, for the qualifying-only fallback case). includeQuali controls
// whether qualifying data is applied to this payload.
//
//nolint:whitespace,funlen // editor/linter issue
func (p *jsonProcessor) buildRacePayload(
	raceSession *SimSession,
	includeQuali bool,
) *processor.ParsedImportPayload {
	hasRace := raceSession != nil
	hasQuali := p.hasQuali && includeQuali
	dataType := detectJSONDataType(hasRace, hasQuali)

	var qualiBestLaps map[string]int
	if includeQuali {
		qualiBestLaps = p.qualiBestLapByResultKey
	}

	var sourceResults []Result
	if hasRace {
		sourceResults = raceSession.Results
	} else {
		sourceResults = p.qualiSession.Results
	}
	offtracks := make(map[int]int)
	if raceSession != nil {
		sessionIncs, ok := p.lapIncidents[raceSession.SimsessionNumber]
		if !ok {
			sessionIncs = []*sessionIncidents{}
		}
		offtracks = p.offtracksByEntry(sessionIncs)
	}

	results := make([]*processor.ResultRow, 0, len(sourceResults))
	for i := range sourceResults {
		source := &sourceResults[i]
		row := p.mapResultRow(source, hasRace, offtracks)
		if ql, ok := qualiBestLaps[resultKey(source)]; ok {
			row.QualiLapTime = ql
		}
		if p.isTeamRace {
			row.TeamDrivers = p.teamDriverIDs.ForResult(source)
		}

		results = append(results, row)
	}

	return &processor.ParsedImportPayload{
		Session: processor.SessionInfo{
			StartTime: p.data.StartTime,
			Track:     formatTrackName(p.data.Track),
		},
		Results:  results,
		DataType: dataType,
	}
}

func (p *jsonProcessor) offtracksByEntry(sessionIncs []*sessionIncidents) map[int]int {
	ret := make(map[int]int)
	for _, si := range sessionIncs {
		for _, inc := range si.Incidents {
			for _, event := range inc.Events {
				if strings.EqualFold(event, "off track") {
					ret[si.RefID]++
				}
			}
		}
	}
	return ret
}

//nolint:whitespace // editor/linter issue
func (p *jsonProcessor) mapResultRow(
	r *Result, hasRace bool, offtracks map[int]int,
) *processor.ResultRow {
	row := &processor.ResultRow{
		CarID:     strconv.Itoa(r.CarID),
		Car:       r.CarName,
		CarNumber: r.Livery.CarNumber,
	}
	if p.isTeamRace {
		// Team races are keyed by team identifiers in result rows.

		row.TeamID = strconv.Itoa(r.TeamID)
		row.Name = r.DisplayName
		row.Offtracks = offtracks[r.TeamID]
	} else {
		row.DriverID = strconv.Itoa(r.CustID)
		row.Name = r.DisplayName
		row.Offtracks = offtracks[r.CustID]
	}

	if hasRace {
		row.FinPos = r.FinishPosition + 1
		row.StartPos = r.StartingPosition + 1
		row.Laps = r.LapsComplete
		row.LapsLed = r.LapsLead
		row.Incidents = r.Incidents
		row.TotalTime = iRacingTimeToMillis(r.AverageLap) * r.LapsComplete
		if r.BestLapTime > 0 {
			row.FastestLapTime = iRacingTimeToMillis(r.BestLapTime)
		}
	} else {
		row.FinPos = r.FinishPosition + 1
		if r.BestLapTime > 0 {
			row.QualiLapTime = iRacingTimeToMillis(r.BestLapTime)
		}
	}

	return row
}

func detectJSONDataType(hasRace, hasQuali bool) processor.ImportData {
	switch {
	case hasRace && hasQuali:
		return processor.ImportDataAll
	case hasRace:
		return processor.ImportDataRace
	case hasQuali:
		return processor.ImportDataQuali
	default:
		return processor.ImportDataAll
	}
}

// findRaceSessions returns all simsessions whose type is SimsessionTypeRace and whose
// type name indicates a race, in the order they appear in session_results. This order
// (rather than the unreliable simsession_number) is used to identify heat races.
//
//nolint:whitespace // editor/linter issue
func findRaceSessions(sessions []SimSession) []SimSession {
	races := make([]SimSession, 0, 1)
	for i := range sessions {
		s := &sessions[i]
		if s.SimsessionType == SimsessionTypeRace &&
			strings.EqualFold(s.SimsessionTypeName, "race") {
			races = append(races, *s)
		}
	}

	return races
}

// findFirstQualifySession returns the first qualifying simsession.
// Open qualifying (type 5) is preferred; lone qualifying (type 6 with a
// non-race type name) is used as a fallback.
func findFirstQualifySession(sessions []SimSession) (SimSession, bool) {
	for i := range sessions {
		s := &sessions[i]
		if isQualifyingSession(s) {
			return *s, true
		}
	}

	return SimSession{}, false
}

func isQualifyingSession(s *SimSession) bool {
	if strings.Contains(strings.ToLower(s.SimsessionTypeName), "qualifying") {
		return true
	}

	switch s.SimsessionType {
	case SimsessionTypeOpenQualifying, SimsessionTypeLoneQualifying:
		return true
	default:
		return false
	}
}

func buildQualiBestLapMap(qualiSession *SimSession, hasQuali bool) map[string]int {
	if !hasQuali {
		return nil
	}

	m := make(map[string]int, len(qualiSession.Results))
	for i := range qualiSession.Results {
		r := &qualiSession.Results[i]
		if r.BestLapTime > 0 {
			m[resultKey(r)] = iRacingTimeToMillis(r.BestLapTime)
		}
	}

	return m
}

type TeamDriverIDs struct {
	byResultKey map[string][]*processor.TeamDriver
}

func (t TeamDriverIDs) ForResult(r *Result) []*processor.TeamDriver {
	if t.byResultKey == nil {
		return nil
	}

	return t.byResultKey[resultKey(r)]
}

//nolint:gocyclo,funlen // aggregation over nested iRacing session/result structures
func collectTeamDriverIDs(sessions []SimSession) TeamDriverIDs {
	entries := make(map[string]map[int]*processor.TeamDriver)

	for i := range sessions {
		session := &sessions[i]
		for j := range session.Results {
			result := &session.Results[j]
			if len(result.DriverResults) == 0 {
				continue
			}

			key := resultKey(result)
			if key == "" {
				continue
			}

			driversByCustID, ok := entries[key]
			if !ok {
				driversByCustID = make(map[int]*processor.TeamDriver)
				entries[key] = driversByCustID
			}

			for k := range result.DriverResults {
				driver := &result.DriverResults[k]
				if driver.CustID == 0 {
					continue
				}

				existing, exists := driversByCustID[driver.CustID]
				if !exists {
					existing = &processor.TeamDriver{DriverID: strconv.Itoa(driver.CustID)}
					driversByCustID[driver.CustID] = existing
				}

				if existing.Name == "" {
					existing.Name = driver.DisplayName
				}
				isBetterBestLap := driver.BestLapTime > 0 &&
					(existing.FastestLapTime == 0 ||
						driver.BestLapTime < existing.FastestLapTime)
				if isBetterBestLap {
					existing.FastestLapTime = iRacingTimeToMillis(driver.BestLapTime)
				}
				if driver.LapsComplete > existing.Laps {
					existing.Laps = driver.LapsComplete
				}
				if driver.Incidents > existing.Incidents {
					existing.Incidents = driver.Incidents
				}
			}
		}
	}

	resolved := make(map[string][]*processor.TeamDriver, len(entries))
	for key, byCustID := range entries {
		drivers := make([]*processor.TeamDriver, 0, len(byCustID))
		for _, driver := range byCustID {
			drivers = append(drivers, driver)
		}

		sort.Slice(drivers, func(i, j int) bool {
			left, leftErr := strconv.Atoi(drivers[i].DriverID)
			right, rightErr := strconv.Atoi(drivers[j].DriverID)
			if leftErr == nil && rightErr == nil {
				return left < right
			}

			return drivers[i].DriverID < drivers[j].DriverID
		})

		resolved[key] = drivers
	}

	return TeamDriverIDs{byResultKey: resolved}
}

func resultKey(r *Result) string {
	if r.TeamID != 0 {
		return "team:" + strconv.Itoa(r.TeamID)
	}
	if r.CustID != 0 {
		return "cust:" + strconv.Itoa(r.CustID)
	}

	displayName := strings.TrimSpace(r.DisplayName)
	if displayName == "" {
		return ""
	}

	return "name:" + strings.ToLower(displayName)
}

func formatTrackName(t Track) string {
	if t.ConfigName != "" && t.ConfigName != "N/A" {
		return t.TrackName + " - " + t.ConfigName
	}

	return t.TrackName
}

// we need millis but iRacing provides microseconds
func iRacingTimeToMillis(iTime int) int {
	return iTime / 10
}

func payloadToEventResult(payload any) (*EventResultEnvelope, error) {
	switch p := payload.(type) {
	case EventResultEnvelope:
		return &p, nil
	case *EventResultEnvelope:
		if p == nil {
			return nil, fmt.Errorf("invalid json payload: nil EventResultEnvelope")
		}

		return p, nil
	case []byte:
		return unmarshalEventResult(p)
	case string:
		return unmarshalEventResult([]byte(p))
	default:
		return nil, fmt.Errorf("unsupported json payload type: %T", payload)
	}
}

func unmarshalEventResult(data []byte) (*EventResultEnvelope, error) {
	var e EventResultEnvelope
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("decode json payload: %w", err)
	}

	return &e, nil
}
