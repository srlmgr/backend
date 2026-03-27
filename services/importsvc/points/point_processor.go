package points

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/samber/lo"

	"github.com/srlmgr/backend/log"
)

//nolint:whitespace // editor/linter issue
func NewPointSystemProcessor(
	ctx context.Context,
	settings *PointSystemSettings,
) *PointSystemProcessor {
	return &PointSystemProcessor{
		logger:   log.GetFromContext(ctx).Named("points"),
		settings: settings,
	}
}

type (
	// TODO: move to model when dev is finished
	Output interface {
		ReferenceID() int32
		Points() PointType
		Msg() string
		Origin() PointPolicyType
	}
	positionBasedPoints interface {
		Process(inputs []Input, points []PointType) []Output
	}

	PointSystemProcessor struct {
		logger   *log.Logger
		settings *PointSystemSettings
	}

	standardPosBasedProcessor struct {
		policyType PointPolicyType
	}

	workOutput struct {
		refID  int32
		points PointType
		msg    string
		origin PointPolicyType
	}
)

var (
	_ positionBasedPoints = (*standardPosBasedProcessor)(nil)
	_ Output              = (*workOutput)(nil)
)

func (w workOutput) ReferenceID() int32      { return w.refID }
func (w workOutput) Points() PointType       { return w.points }
func (w workOutput) Msg() string             { return w.msg }
func (w workOutput) Origin() PointPolicyType { return w.origin }

//nolint:whitespace // editor/linter issue
func (p *standardPosBasedProcessor) Process(
	inputs []Input, points []PointType,
) []Output {
	ret := make([]Output, 0)
	for i, input := range inputs {
		if i >= len(points) {
			break
		}
		var policy PointPolicy
		msg := fmt.Sprintf("%s for pos %d", policy, i+1)

		ret = append(ret, workOutput{
			refID:  input.ReferenceID(),
			points: points[i],
			msg:    msg,
			origin: p.policyType,
		})
	}
	return ret
}

//nolint:whitespace,funlen // editor/linter issue
func (p *PointSystemProcessor) ProcessPoints(
	inputs []Input,
	policies []PointPolicyType,
	awardSettings map[PointPolicyType][]PointType,
	penaltySettings map[PointPolicyType]any,
) ([]Output, error) {
	ret := make([]Output, 0)
	byClass := lo.GroupBy(inputs, func(item Input) int32 { return item.ClassID() })
	for _, classInputs := range byClass {
		eligibleInputs := p.collectEligibleInputs(classInputs)
		for _, policyType := range policies {
			//nolint:exhaustive // only position based here
			switch policyType {
			case PointsPolicyFinishPos, PointsPolicyTopNFinishers:
				worker := &standardPosBasedProcessor{policyType: policyType}
				pInput := lo.Clone(eligibleInputs)
				slices.SortFunc(
					pInput, func(a, b Input) int {
						return int(a.FinishPosition() - b.FinishPosition())
					})

				polSettings := awardSettings[policyType]
				ret = append(ret, worker.Process(pInput, polSettings)...)
			case PointsPolicyQualificationPos:
				worker := &standardPosBasedProcessor{policyType: policyType}
				pInput := lo.Clone(eligibleInputs)
				slices.SortFunc(
					pInput, func(a, b Input) int {
						return int(a.QualiPosition() - b.QualiPosition())
					})
				polSettings := awardSettings[policyType]
				ret = append(ret, worker.Process(pInput, polSettings)...)

			case PointsPolicyFastestLap:
				worker := &standardPosBasedProcessor{policyType: policyType}
				pInput := lo.Clone(eligibleInputs)
				slices.SortFunc(
					pInput, func(a, b Input) int {
						return int(a.FastestLap() - b.FastestLap())
					})
				polSettings := awardSettings[policyType]
				ret = append(ret, worker.Process(pInput, polSettings)...)

			case PointsPolicyLeastIncidents:
				worker := &standardPosBasedProcessor{policyType: policyType}
				pInput := lo.Clone(eligibleInputs)
				slices.SortFunc(
					pInput, func(a, b Input) int {
						return int(a.Incidents() - b.Incidents())
					})
				polSettings := awardSettings[policyType]
				ret = append(ret, worker.Process(pInput, polSettings)...)
			}
		}
		// next: apply policies on awarded position points
		// e.g. stuff like "10% reduction for more than 3 incidents"
		for _, policyType := range policies {
			//nolint:exhaustive,gocritic // may be extended with addition policies
			switch policyType {
			case PointsPolicyIncidentsExceeded:
				polSettings, ok := penaltySettings[policyType].(ThresholdPenaltySettings)
				if !ok {
					continue
				}
				ret = append(
					ret,
					p.handleIncidentsExceededPolicy(ret, polSettings, eligibleInputs)...)
			}
		}
	}

	return ret, nil
}

// uses already produced outputs from previous steps
func (p *PointSystemProcessor) handleIncidentsExceededPolicy(
	outputs []Output, settings ThresholdPenaltySettings, inputs []Input,
) []Output {
	// here we are interested in output produced by PointsPolicyFinishPos
	filtered := lo.Filter(outputs, func(output Output, _ int) bool {
		return output.Origin() == PointsPolicyFinishPos
	})
	byRef := lo.SliceToMap(filtered, func(item Output) (int32, PointType) {
		return item.ReferenceID(), item.Points()
	})
	ret := make([]Output, 0)
	for _, inp := range inputs {
		// TODO: need to figure out how handle different policy type settings
		// and how to extract the parameter values
		// here we need: incidentThreshold, reductionPct, and maybe others in the future
		threshold := settings.Threshold
		penaltyPct := settings.PenaltyPct
		if inp.Incidents() > threshold {
			refID := inp.ReferenceID()
			points, ok := byRef[refID]
			if ok {
				out := workOutput{
					refID:  refID,
					points: PointType(math.Round(float64(points) * -penaltyPct)),
					msg: fmt.Sprintf("%d%% reduction for %d incidents (limit: %d)",
						int(penaltyPct*100), inp.Incidents(), threshold),
					origin: PointsPolicyIncidentsExceeded,
				}
				ret = append(ret, out)
			}
		}
	}
	return ret
}

func (p *PointSystemProcessor) collectEligibleInputs(inputs []Input) []Input {
	first, _ := lo.Find(inputs, func(inp Input) bool {
		return inp.FinishPosition() == 1
	})
	needsLaps := int32(0)
	if p.settings.Eligibility.RaceDistPct > 0 {
		raw := float64(first.LapsCompleted()) * p.settings.Eligibility.RaceDistPct
		needsLaps = int32(math.Ceil(raw))
	}
	ret := lo.Filter(inputs, func(inp Input, _ int) bool {
		if !p.settings.Eligibility.Guests && inp.IsGuest() {
			return false
		}
		if needsLaps > 0 && inp.LapsCompleted() < needsLaps {
			return false
		}
		return true
	})
	return ret
}
