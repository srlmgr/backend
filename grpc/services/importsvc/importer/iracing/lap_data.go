package iracing

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/srlmgr/backend/log"
	gUtil "github.com/srlmgr/backend/support"
	"github.com/srlmgr/backend/support/iracing/irdata"
)

type (
	lapDataCollector struct {
		api          *irdata.IrData
		l            *log.Logger
		subsessionID int
	}
	incidentData struct {
		LapNumber int
		Events    []string
	}
	sessionIncidents struct {
		RefID     int
		Name      string
		Incidents []*incidentData
	}

	incsBySession map[int][]*sessionIncidents
)

func newLapDataCollector(api *irdata.IrData) *lapDataCollector {
	return &lapDataCollector{
		api:          api,
		l:            log.Default().Named("importer.iracing.lap_data"),
		subsessionID: 0,
	}
}

//nolint:whitespace // editor/linter issue
func (c *lapDataCollector) run(
	eventResult *EventResult,
) incsBySession {
	ret := make(incsBySession)
	c.subsessionID = eventResult.SubsessionID
	for i := range eventResult.SessionResults {
		sResults := eventResult.SessionResults[i]
		if sResults.SimsessionTypeName == "Race" {
			incidents := c.incCollector(
				eventResult.MaxTeamDrivers > 1,
				sResults.SimsessionNumber,
				sResults.Results,
			)
			ret[sResults.SimsessionNumber] = incidents
		}
	}
	return ret
}

//nolint:whitespace // editor/linter issue
func (c *lapDataCollector) incCollector(
	isTeamRace bool,
	sessionNumber int,
	resultEntries []Result,
) []*sessionIncidents {
	result := []*sessionIncidents{}
	mu := sync.Mutex{}

	worker := gUtil.NewWorker[Result, sessionIncidents](
		func(entry Result) (sessionIncidents, error) {
			incs, err := c.collectLapData(isTeamRace, sessionNumber, &entry)
			if err != nil {
				c.l.Error("failed to collect lap data", log.ErrorField(err))
				return sessionIncidents{}, err
			}
			id := entry.CustID
			if isTeamRace {
				id = entry.TeamID
			}
			return sessionIncidents{
				RefID:     id,
				Name:      entry.DisplayName,
				Incidents: incs,
			}, nil
		}, gUtil.WithResultCallback(func(idx int, si sessionIncidents, err error) {
			if err != nil {
				return
			}
			mu.Lock()
			result = append(result, &si)
			mu.Unlock()
		}),
		gUtil.WithNumWorker[sessionIncidents](5),
	)
	worker.Process(resultEntries)
	return result
}

//nolint:whitespace // editor/linter issue
func (c *lapDataCollector) collectLapData(
	isTeamRace bool,
	sessionNumber int, resultEntry *Result,
) ([]*incidentData, error) {
	v := url.Values{}
	v.Set("subsession_id", fmt.Sprintf("%d", c.subsessionID))
	v.Set("simsession_number", fmt.Sprintf("%d", sessionNumber))

	if isTeamRace {
		v.Set("team_id", fmt.Sprintf("%d", resultEntry.TeamID))
	} else {
		v.Set("cust_id", fmt.Sprintf("%d", resultEntry.CustID))
	}
	data, err := c.api.Get(
		strings.TrimSpace(fmt.Sprintf(`/data/results/lap_data?%s`, v.Encode())),
	)
	if err != nil {
		c.l.Error("failed to get lap data", log.ErrorField(err))
		return nil, err
	}

	var chunkedData irdata.ChunkData[LapData]
	err = json.Unmarshal(data, &chunkedData)
	if err != nil {
		c.l.Error("failed to parse lap data", log.ErrorField(err))
		return nil, err
	}
	var incidents []*incidentData
	for i := range chunkedData.Data {
		lapData := chunkedData.Data[i]
		if len(lapData.LapEvents) > 0 {
			c.l.Debug("lap data has events",
				log.Int("LapNo", lapData.LapNumber),
				log.Any("events", lapData.LapEvents))
			incidents = append(incidents, &incidentData{
				LapNumber: lapData.LapNumber,
				Events:    lapData.LapEvents,
			})
		}
	}
	return incidents, nil
}
