package points

import (
	"context"
	"math"
)

type (
	EventProcessor struct {
		settings *PointSystemSettings
	}
	GridInput struct {
		GridID int32
		Inputs []Input
	}
	GridOutput struct {
		GridID  int32
		Outputs []Output
	}
	// ResolveGridID resolves a gridID to a raceNo and gridNo.
	// This is needed to determine which policies to apply for the given grid.
	// NOTE: raceNo and gridNo are 0-based.
	ResolveGridID func(gridID int32) (raceNo, gridNo int32, err error)
)

//nolint:whitespace // editor/linter issue
func NewEventProcessor(
	settings *PointSystemSettings,
) *EventProcessor {
	return &EventProcessor{
		settings: settings,
	}
}

//nolint:whitespace // editor/linter issue
func (p *EventProcessor) ProcessAll(
	ctx context.Context,
	allInputs []GridInput,
	resolver ResolveGridID) (
	[]GridOutput, error,
) {
	outputs := make([]GridOutput, 0)
	for i := range allInputs {
		grid := allInputs[i]
		raceNo, gridNo, err := resolver(grid.GridID)
		if err != nil {
			return nil, err
		}
		raceNo = int32(math.Min(float64(raceNo), float64(len(p.settings.Races)-1)))
		raceSettings := p.settings.Races[raceNo]

		gridNo = int32(math.Min(float64(gridNo), float64(len(raceSettings.AwardSettings)-1)))
		pointProc := NewPointSystemProcessor(
			ctx,
			p.settings)
		gridOutputs, err := pointProc.ProcessPoints(
			grid.Inputs,
			raceSettings.Policies,
			raceSettings.AwardSettings[gridNo].Points,
			raceSettings.PenaltySettings[gridNo].Arguments,
		)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, GridOutput{
			GridID:  grid.GridID,
			Outputs: gridOutputs,
		})
	}
	return outputs, nil
}
