//nolint:lll,funlen // complex test setup
package points

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/srlmgr/backend/log"
)

func TestNewEventProcessor(t *testing.T) {
	t.Parallel()

	settings := &PointSystemSettings{}
	processor := NewEventProcessor(settings)

	if processor == nil {
		t.Fatal("expected non-nil processor")
	}
	if processor.settings != settings {
		t.Fatal("expected processor to keep provided settings pointer")
	}
}

func TestEventProcessorProcessAll_ProcessesEachGridWithResolvedRaceAndGrid(t *testing.T) {
	t.Parallel()

	settings := &PointSystemSettings{
		Eligibility: EligibilitySettings{},
		Races: []RaceSettings{
			{
				Policies: []PointPolicyType{PointsPolicyFinishPos},
				AwardSettings: []RankedPolicySettings{
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {10, 7},
						},
					},
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {5, 3},
						},
					},
				},
				PenaltySettings: []PointPenaltySettings{
					{Arguments: map[PointPolicyType]any{}},
					{Arguments: map[PointPolicyType]any{}},
				},
			},
			{
				Policies: []PointPolicyType{PointsPolicyFinishPos},
				AwardSettings: []RankedPolicySettings{
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {20, 15},
						},
					},
				},
				PenaltySettings: []PointPenaltySettings{
					{Arguments: map[PointPolicyType]any{}},
				},
			},
		},
	}

	processor := NewEventProcessor(settings)
	ctx := log.AddToContext(context.Background(), log.New())

	inputs := []GridInput{
		{
			GridID: 1001,
			Inputs: []Input{
				NewInput(
					WithClassID(1),
					WithReferenceID(11),
					WithFinishPosition(2),
					WithLapsCompleted(10),
				),
				NewInput(
					WithClassID(1),
					WithReferenceID(12),
					WithFinishPosition(1),
					WithLapsCompleted(10),
				),
			},
		},
		{
			GridID: 2002,
			Inputs: []Input{
				NewInput(
					WithClassID(2),
					WithReferenceID(21),
					WithFinishPosition(1),
					WithLapsCompleted(10),
				),
				NewInput(
					WithClassID(2),
					WithReferenceID(22),
					WithFinishPosition(2),
					WithLapsCompleted(10),
				),
			},
		},
	}

	resolver := func(gridID int32) (int32, int32, error) {
		switch gridID {
		case 1001:
			return 0, 1, nil
		case 2002:
			return 1, 0, nil
		default:
			return 0, 0, fmt.Errorf("unexpected grid id: %d", gridID)
		}
	}

	outputs, err := processor.ProcessAll(ctx, inputs, resolver)
	if err != nil {
		t.Fatalf("ProcessAll returned unexpected error: %v", err)
	}

	if len(outputs) != 2 {
		t.Fatalf("unexpected outputs length: got %d want 2", len(outputs))
	}

	if outputs[0].GridID != 1001 {
		t.Fatalf("unexpected grid id at index 0: got %d want 1001", outputs[0].GridID)
	}
	if outputs[1].GridID != 2002 {
		t.Fatalf("unexpected grid id at index 1: got %d want 2002", outputs[1].GridID)
	}

	expectedFirst := []Output{
		workOutput{
			refID:  12,
			points: 5,
			msg:    fmt.Sprintf("for pos %d", 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  11,
			points: 3,
			msg:    fmt.Sprintf("for pos %d", 2),
			origin: PointsPolicyFinishPos,
		},
	}

	expectedSecond := []Output{
		workOutput{
			refID:  21,
			points: 20,
			msg:    fmt.Sprintf("for pos %d", 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  22,
			points: 15,
			msg:    fmt.Sprintf("for pos %d", 2),
			origin: PointsPolicyFinishPos,
		},
	}

	assertOutputSlicesEqual(t, outputs[0].Outputs, expectedFirst)
	assertOutputSlicesEqual(t, outputs[1].Outputs, expectedSecond)
}

func TestEventProcessorProcessAll_ClampsResolvedIndexesToLastConfiguredRaceAndGrid(t *testing.T) {
	t.Parallel()

	settings := &PointSystemSettings{
		Eligibility: EligibilitySettings{},
		Races: []RaceSettings{
			{
				Policies: []PointPolicyType{PointsPolicyFinishPos},
				AwardSettings: []RankedPolicySettings{
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {3, 2},
						},
					},
				},
				PenaltySettings: []PointPenaltySettings{
					{Arguments: map[PointPolicyType]any{}},
				},
			},
			{
				Policies: []PointPolicyType{PointsPolicyFinishPos},
				AwardSettings: []RankedPolicySettings{
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {6, 4},
						},
					},
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {50, 40},
						},
					},
				},
				PenaltySettings: []PointPenaltySettings{
					{Arguments: map[PointPolicyType]any{}},
					{Arguments: map[PointPolicyType]any{}},
				},
			},
		},
	}

	processor := NewEventProcessor(settings)
	ctx := log.AddToContext(context.Background(), log.New())

	inputs := []GridInput{
		{
			GridID: 77,
			Inputs: []Input{
				NewInput(
					WithClassID(1),
					WithReferenceID(701),
					WithFinishPosition(2),
					WithLapsCompleted(5),
				),
				NewInput(
					WithClassID(1),
					WithReferenceID(702),
					WithFinishPosition(1),
					WithLapsCompleted(5),
				),
			},
		},
	}

	resolver := func(gridID int32) (int32, int32, error) {
		if gridID != 77 {
			return 0, 0, fmt.Errorf("unexpected grid id: %d", gridID)
		}
		return 99, 88, nil
	}

	outputs, err := processor.ProcessAll(ctx, inputs, resolver)
	if err != nil {
		t.Fatalf("ProcessAll returned unexpected error: %v", err)
	}

	if len(outputs) != 1 {
		t.Fatalf("unexpected outputs length: got %d want 1", len(outputs))
	}

	expected := []Output{
		workOutput{
			refID:  702,
			points: 50,
			msg:    fmt.Sprintf("for pos %d", 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  701,
			points: 40,
			msg:    fmt.Sprintf("for pos %d", 2),
			origin: PointsPolicyFinishPos,
		},
	}

	assertOutputSlicesEqual(t, outputs[0].Outputs, expected)
}

func TestEventProcessorProcessAll_WhenRaceNoIsBeyondConfigured_UsesLastRaceSettings(t *testing.T) {
	t.Parallel()

	settings := &PointSystemSettings{
		Eligibility: EligibilitySettings{},
		Races: []RaceSettings{
			{
				Policies: []PointPolicyType{PointsPolicyFinishPos},
				AwardSettings: []RankedPolicySettings{
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {8, 6},
						},
					},
				},
				PenaltySettings: []PointPenaltySettings{{Arguments: map[PointPolicyType]any{}}},
			},
			{
				Policies: []PointPolicyType{PointsPolicyFinishPos},
				AwardSettings: []RankedPolicySettings{
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {30, 20},
						},
					},
				},
				PenaltySettings: []PointPenaltySettings{{Arguments: map[PointPolicyType]any{}}},
			},
		},
	}

	processor := NewEventProcessor(settings)
	ctx := log.AddToContext(context.Background(), log.New())

	inputs := []GridInput{
		{
			GridID: 301,
			Inputs: []Input{
				NewInput(
					WithClassID(1),
					WithReferenceID(9001),
					WithFinishPosition(2),
					WithLapsCompleted(20),
				),
				NewInput(
					WithClassID(1),
					WithReferenceID(9002),
					WithFinishPosition(1),
					WithLapsCompleted(20),
				),
			},
		},
	}

	resolver := func(gridID int32) (int32, int32, error) {
		if gridID != 301 {
			return 0, 0, fmt.Errorf("unexpected grid id: %d", gridID)
		}
		return 999, 0, nil
	}

	outputs, err := processor.ProcessAll(ctx, inputs, resolver)
	if err != nil {
		t.Fatalf("ProcessAll returned unexpected error: %v", err)
	}

	if len(outputs) != 1 {
		t.Fatalf("unexpected outputs length: got %d want 1", len(outputs))
	}

	expected := []Output{
		workOutput{
			refID:  9002,
			points: 30,
			msg:    fmt.Sprintf("for pos %d", 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  9001,
			points: 20,
			msg:    fmt.Sprintf("for pos %d", 2),
			origin: PointsPolicyFinishPos,
		},
	}

	assertOutputSlicesEqual(t, outputs[0].Outputs, expected)
}

//nolint:whitespace // editor/linter issue
func TestEventProcessorProcessAll_WhenGridNoIsBeyondConfigured_UsesLastGridAwardAndPenaltySettings(
	t *testing.T,
) {
	t.Parallel()

	settings := &PointSystemSettings{
		Eligibility: EligibilitySettings{},
		Races: []RaceSettings{
			{
				Policies: []PointPolicyType{PointsPolicyFinishPos, PointsPolicyIncidentsExceeded},
				AwardSettings: []RankedPolicySettings{
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {11, 9},
						},
					},
					{
						Points: map[PointPolicyType][]PointType{
							PointsPolicyFinishPos: {100, 70},
						},
					},
				},
				PenaltySettings: []PointPenaltySettings{
					{
						Arguments: map[PointPolicyType]any{
							PointsPolicyIncidentsExceeded: ThresholdPenaltySettings{
								Threshold:  3,
								PenaltyPct: 0.1,
							},
						},
					},
					{
						Arguments: map[PointPolicyType]any{
							PointsPolicyIncidentsExceeded: ThresholdPenaltySettings{
								Threshold:  1,
								PenaltyPct: 0.5,
							},
						},
					},
				},
			},
		},
	}

	processor := NewEventProcessor(settings)
	ctx := log.AddToContext(context.Background(), log.New())

	inputs := []GridInput{
		{
			GridID: 302,
			Inputs: []Input{
				NewInput(
					WithClassID(1),
					WithReferenceID(9101),
					WithFinishPosition(1),
					WithLapsCompleted(20),
					WithIncidents(0),
				),
				NewInput(
					WithClassID(1),
					WithReferenceID(9102),
					WithFinishPosition(2),
					WithLapsCompleted(20),
					WithIncidents(3),
				),
			},
		},
	}

	resolver := func(gridID int32) (int32, int32, error) {
		if gridID != 302 {
			return 0, 0, fmt.Errorf("unexpected grid id: %d", gridID)
		}
		return 0, 99, nil
	}

	outputs, err := processor.ProcessAll(ctx, inputs, resolver)
	if err != nil {
		t.Fatalf("ProcessAll returned unexpected error: %v", err)
	}

	if len(outputs) != 1 {
		t.Fatalf("unexpected outputs length: got %d want 1", len(outputs))
	}

	expected := []Output{
		workOutput{
			refID:  9101,
			points: 100,
			msg:    fmt.Sprintf("for pos %d", 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  9102,
			points: 70,
			msg:    fmt.Sprintf("for pos %d", 2),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  9102,
			points: -35,
			msg:    "50% reduction for 3 incidents (limit: 1)",
			origin: PointsPolicyIncidentsExceeded,
		},
	}

	assertOutputSlicesEqual(t, outputs[0].Outputs, expected)
}

func TestEventProcessorProcessAll_ReturnsResolverError(t *testing.T) {
	t.Parallel()

	settings := &PointSystemSettings{
		Eligibility: EligibilitySettings{},
		Races: []RaceSettings{
			{
				Policies: []PointPolicyType{PointsPolicyFinishPos},
				AwardSettings: []RankedPolicySettings{
					{Points: map[PointPolicyType][]PointType{PointsPolicyFinishPos: {1}}},
				},
				PenaltySettings: []PointPenaltySettings{{Arguments: map[PointPolicyType]any{}}},
			},
		},
	}

	processor := NewEventProcessor(settings)
	ctx := log.AddToContext(context.Background(), log.New())

	wantErr := errors.New("resolver failed")
	resolver := func(gridID int32) (int32, int32, error) {
		if gridID != 123 {
			return 0, 0, fmt.Errorf("unexpected grid id: %d", gridID)
		}
		return 0, 0, wantErr
	}

	outputs, err := processor.ProcessAll(
		ctx,
		[]GridInput{{
			GridID: 123,
			Inputs: []Input{
				NewInput(
					WithClassID(1),
					WithReferenceID(1),
					WithFinishPosition(1),
					WithLapsCompleted(1),
				),
			},
		}},
		resolver,
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected error: got %v want %v", err, wantErr)
	}
	if outputs != nil {
		t.Fatalf("expected nil outputs on resolver error, got length %d", len(outputs))
	}
}
