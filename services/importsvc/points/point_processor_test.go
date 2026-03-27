//nolint:lll,funlen // complex test setup
package points

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

type outputSnapshot struct {
	refID  int32
	points PointType
	msg    string
	origin PointPolicyType
}

func toOutputSnapshots(outputs []Output) []outputSnapshot {
	ret := make([]outputSnapshot, 0, len(outputs))
	for _, out := range outputs {
		ret = append(ret, outputSnapshot{
			refID:  out.ReferenceID(),
			points: out.Points(),
			msg:    out.Msg(),
			origin: out.Origin(),
		})
	}
	return ret
}

func assertOutputSlicesEqual(t *testing.T, got, want []Output) {
	t.Helper()

	gotSnapshots := toOutputSnapshots(got)
	wantSnapshots := toOutputSnapshots(want)

	if len(gotSnapshots) != len(wantSnapshots) {
		t.Fatalf("unexpected output length: got %d want %d", len(gotSnapshots), len(wantSnapshots))
	}

	for i := range gotSnapshots {
		if gotSnapshots[i] != wantSnapshots[i] {
			t.Fatalf(
				"unexpected output at index %d: got %+v want %+v",
				i,
				gotSnapshots[i],
				wantSnapshots[i],
			)
		}
	}
}

func assertOutputSlicesEqualUnordered(t *testing.T, got, want []Output) {
	t.Helper()

	gotSnapshots := toOutputSnapshots(got)
	wantSnapshots := toOutputSnapshots(want)

	sortFn := func(a, b outputSnapshot) int {
		if a.refID != b.refID {
			return int(a.refID - b.refID)
		}
		if a.origin != b.origin {
			return int(a.origin - b.origin)
		}
		if a.points != b.points {
			if a.points < b.points {
				return -1
			}
			return 1
		}
		return strings.Compare(a.msg, b.msg)
	}

	slices.SortFunc(gotSnapshots, sortFn)
	slices.SortFunc(wantSnapshots, sortFn)

	if len(gotSnapshots) != len(wantSnapshots) {
		t.Fatalf("unexpected output length: got %d want %d", len(gotSnapshots), len(wantSnapshots))
	}

	for i := range gotSnapshots {
		if gotSnapshots[i] != wantSnapshots[i] {
			t.Fatalf(
				"unexpected output at index %d: got %+v want %+v",
				i,
				gotSnapshots[i],
				wantSnapshots[i],
			)
		}
	}
}

func TestStandardPosBasedProcessorProcess_MapsByInputOrder(t *testing.T) {
	t.Parallel()

	processor := &standardPosBasedProcessor{policyType: PointsPolicyFinishPos}
	inputs := []Input{
		NewInput(WithReferenceID(101)),
		NewInput(WithReferenceID(202)),
	}
	points := []PointType{25, 18}

	outputs := processor.Process(inputs, points)

	if len(outputs) != 2 {
		t.Fatalf("unexpected output length: got %d want 2", len(outputs))
	}

	if outputs[0].ReferenceID() != 101 {
		t.Fatalf("unexpected ref id at index 0: got %d want 101", outputs[0].ReferenceID())
	}
	if outputs[0].Points() != 25 {
		t.Fatalf("unexpected points at index 0: got %v want 25", outputs[0].Points())
	}
	if outputs[0].Origin() != PointsPolicyFinishPos {
		t.Fatalf(
			"unexpected origin at index 0: got %v want %v",
			outputs[0].Origin(),
			PointsPolicyFinishPos,
		)
	}
	if !strings.HasSuffix(outputs[0].Msg(), "for pos 1") {
		t.Fatalf("unexpected msg at index 0: got %q", outputs[0].Msg())
	}

	if outputs[1].ReferenceID() != 202 {
		t.Fatalf("unexpected ref id at index 1: got %d want 202", outputs[1].ReferenceID())
	}
	if outputs[1].Points() != 18 {
		t.Fatalf("unexpected points at index 1: got %v want 18", outputs[1].Points())
	}
	if outputs[1].Origin() != PointsPolicyFinishPos {
		t.Fatalf(
			"unexpected origin at index 1: got %v want %v",
			outputs[1].Origin(),
			PointsPolicyFinishPos,
		)
	}
	if !strings.HasSuffix(outputs[1].Msg(), "for pos 2") {
		t.Fatalf("unexpected msg at index 1: got %q", outputs[1].Msg())
	}
}

func TestStandardPosBasedProcessorProcess_TruncatesWhenPointsAreShorter(t *testing.T) {
	t.Parallel()

	processor := &standardPosBasedProcessor{policyType: PointsPolicyTopNFinishers}
	inputs := []Input{
		NewInput(WithReferenceID(1)),
		NewInput(WithReferenceID(2)),
		NewInput(WithReferenceID(3)),
	}
	points := []PointType{10, 5}

	outputs := processor.Process(inputs, points)

	if len(outputs) != 2 {
		t.Fatalf("unexpected output length: got %d want 2", len(outputs))
	}

	if outputs[0].ReferenceID() != 1 || outputs[1].ReferenceID() != 2 {
		t.Fatalf(
			"unexpected processed ref ids: got [%d, %d] want [1, 2]",
			outputs[0].ReferenceID(),
			outputs[1].ReferenceID(),
		)
	}
}

func TestStandardPosBasedProcessorProcess_EmptyInputs(t *testing.T) {
	t.Parallel()

	processor := &standardPosBasedProcessor{policyType: PointsPolicyFastestLap}
	outputs := processor.Process(nil, []PointType{1})

	if len(outputs) != 0 {
		t.Fatalf("unexpected output length: got %d want 0", len(outputs))
	}
}

func TestPointSystemProcessorProcessPoints_FinishPolicySortsByFinishPosition(t *testing.T) {
	t.Parallel()

	processor := &PointSystemProcessor{
		settings: &PointSystemSettings{
			Eligibility: EligibilitySettings{},
		},
	}

	inputs := []Input{
		NewInput(
			WithClassID(1),
			WithReferenceID(101),
			WithFinishPosition(3),
			WithLapsCompleted(100),
		),
		NewInput(
			WithClassID(1),
			WithReferenceID(102),
			WithFinishPosition(1),
			WithLapsCompleted(100),
		),
		NewInput(
			WithClassID(1),
			WithReferenceID(103),
			WithFinishPosition(2),
			WithLapsCompleted(100),
		),
	}

	awards := map[PointPolicyType][]PointType{
		PointsPolicyFinishPos: {25, 18, 15},
	}

	outputs, err := processor.ProcessPoints(
		inputs,
		[]PointPolicyType{PointsPolicyFinishPos},
		awards,
		nil,
	)
	if err != nil {
		t.Fatalf("ProcessPoints returned unexpected error: %v", err)
	}

	expected := []Output{
		workOutput{
			refID:  102,
			points: 25,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  103,
			points: 18,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 2),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  101,
			points: 15,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 3),
			origin: PointsPolicyFinishPos,
		},
	}

	assertOutputSlicesEqual(t, outputs, expected)
}

func TestPointSystemProcessorProcessPoints_AppliesEligibilityBeforePolicies(t *testing.T) {
	t.Parallel()

	processor := &PointSystemProcessor{
		settings: &PointSystemSettings{
			Eligibility: EligibilitySettings{
				Guests:      false,
				RaceDistPct: 0.5,
			},
		},
	}

	inputs := []Input{
		NewInput(
			WithClassID(1),
			WithReferenceID(201),
			WithFinishPosition(1),
			WithQualiPosition(2),
			WithLapsCompleted(100),
		),
		NewInput(
			WithClassID(1),
			WithReferenceID(202),
			WithFinishPosition(2),
			WithQualiPosition(1),
			WithLapsCompleted(100),
			WithIsGuest(true),
		),
		NewInput(
			WithClassID(1),
			WithReferenceID(203),
			WithFinishPosition(3),
			WithQualiPosition(3),
			WithLapsCompleted(40),
		),
	}

	awards := map[PointPolicyType][]PointType{
		PointsPolicyFinishPos:        {25, 18, 15},
		PointsPolicyQualificationPos: {5, 3, 1},
	}

	outputs, err := processor.ProcessPoints(
		inputs,
		[]PointPolicyType{PointsPolicyFinishPos, PointsPolicyQualificationPos},
		awards,
		nil,
	)
	if err != nil {
		t.Fatalf("ProcessPoints returned unexpected error: %v", err)
	}

	expected := []Output{
		workOutput{
			refID:  201,
			points: 25,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  201,
			points: 5,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 1),
			origin: PointsPolicyQualificationPos,
		},
	}

	assertOutputSlicesEqual(t, outputs, expected)
}

func TestPointSystemProcessorProcessPoints_MultipleClassesProcessIndependently(t *testing.T) {
	t.Parallel()

	processor := &PointSystemProcessor{
		settings: &PointSystemSettings{
			Eligibility: EligibilitySettings{},
		},
	}

	inputs := []Input{
		NewInput(
			WithClassID(1),
			WithReferenceID(301),
			WithFinishPosition(2),
			WithLapsCompleted(10),
		),
		NewInput(
			WithClassID(1),
			WithReferenceID(302),
			WithFinishPosition(1),
			WithLapsCompleted(10),
		),
		NewInput(
			WithClassID(2),
			WithReferenceID(401),
			WithFinishPosition(1),
			WithLapsCompleted(10),
		),
		NewInput(
			WithClassID(2),
			WithReferenceID(402),
			WithFinishPosition(2),
			WithLapsCompleted(10),
		),
	}

	awards := map[PointPolicyType][]PointType{
		PointsPolicyFinishPos: {10, 7},
	}

	outputs, err := processor.ProcessPoints(
		inputs,
		[]PointPolicyType{PointsPolicyFinishPos},
		awards,
		nil,
	)
	if err != nil {
		t.Fatalf("ProcessPoints returned unexpected error: %v", err)
	}

	expected := []Output{
		workOutput{
			refID:  302,
			points: 10,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  301,
			points: 7,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 2),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  401,
			points: 10,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  402,
			points: 7,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 2),
			origin: PointsPolicyFinishPos,
		},
	}

	assertOutputSlicesEqualUnordered(t, outputs, expected)
}

func TestPointSystemProcessorProcessPoints_AppliesIncidentPenaltySettings(t *testing.T) {
	t.Parallel()

	processor := &PointSystemProcessor{
		settings: &PointSystemSettings{
			Eligibility: EligibilitySettings{},
		},
	}

	inputs := []Input{
		NewInput(
			WithClassID(1),
			WithReferenceID(501),
			WithFinishPosition(1),
			WithLapsCompleted(20),
			WithIncidents(2),
		),
		NewInput(
			WithClassID(1),
			WithReferenceID(502),
			WithFinishPosition(2),
			WithLapsCompleted(20),
			WithIncidents(5),
		),
	}

	awards := map[PointPolicyType][]PointType{
		PointsPolicyFinishPos: {25, 18},
	}
	penalties := map[PointPolicyType]PointPenaltySettings{
		PointsPolicyIncidentsExceeded: {
			Name:       "incident penalty",
			Threshold:  3,
			PenaltyPct: 0.1,
		},
	}

	outputs, err := processor.ProcessPoints(
		inputs,
		[]PointPolicyType{PointsPolicyFinishPos, PointsPolicyIncidentsExceeded},
		awards,
		penalties,
	)
	if err != nil {
		t.Fatalf("ProcessPoints returned unexpected error: %v", err)
	}

	expected := []Output{
		workOutput{
			refID:  501,
			points: 25,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 1),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  502,
			points: 18,
			msg:    fmt.Sprintf("%s for pos %d", PointPolicy(nil), 2),
			origin: PointsPolicyFinishPos,
		},
		workOutput{
			refID:  502,
			points: -2,
			msg:    "10% reduction for 5 incidents (limit: 3)",
			origin: PointsPolicyIncidentsExceeded,
		},
	}

	assertOutputSlicesEqual(t, outputs, expected)
}
