package points

import (
	"github.com/srlmgr/backend/db/models"
)

type (
	Converter struct{}
)

func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) ResultEntryToInput(re *models.ResultEntry, opts ...InputOpt) Input {
	standardOps := []InputOpt{
		WithDriverID(re.DriverID.GetOrZero()),
		WithTeamID(re.TeamID.GetOrZero()),
		WithTeamDriverIDs(re.TeamDrivers.GetOrZero().DriverIDs),
		WithClassID(re.CarClassID.GetOrZero()),
		WithFinishPosition(re.FinishPosition),
		WithQualiPosition(re.StartPosition.GetOrZero()),
		WithIsGuest(re.IsGuestStarter),
		WithIncidents(re.Incidents.GetOrZero()),
		WithLapsCompleted(re.LapsCompleted),
		WithFastestLap(re.FastestLapTimeMS.GetOrZero()),
	}
	standardOps = append(standardOps, opts...)
	return NewInput(standardOps...)
}
