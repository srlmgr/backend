package points

import "github.com/srlmgr/backend/db/models"

type (
	Converter struct{}
)

func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) ResultEntryToInput(re *models.ResultEntry) Input {
	return NewInput(
		WithDriverID(re.DriverID.GetOrZero()),
		WithTeamID(re.TeamID.GetOrZero()),
		WithClassID(re.CarClassID.GetOrZero()),
		WithFinishPosition(re.FinishingPosition),
		WithQualiPosition(re.StartingPosition.GetOrZero()),
		WithIsGuest(re.IsGuestDriver),
		WithIncidents(re.Incidents.GetOrZero()),
		WithLapsCompleted(re.CompletedLaps),
		WithFastestLap(re.FastestLapTimeMS.GetOrZero()),
	)
}
