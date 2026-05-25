package query

import (
	"reflect"
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
)

func TestSummaryReferenceIDs(t *testing.T) {
	input := []*commonv1.Summary{
		{ReferenceId: 9},
		nil,
		{ReferenceId: 3},
		{ReferenceId: 9},
		{ReferenceId: 0},
		{ReferenceId: 7},
	}

	got := summaryReferenceIDs(input)
	want := []int32{3, 7, 9}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summaryReferenceIDs() = %v, want %v", got, want)
	}
}
