//nolint:lll // service tests can have long lines for test data setup
package conversion

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/lib/pq"

	"github.com/srlmgr/backend/db/models"
)

func TestServiceRacingSimToSimulation(t *testing.T) {
	t.Parallel()

	frontendID := uuid.Must(uuid.NewV4())
	input := &models.RacingSim{
		ID:                     42,
		FrontendID:             frontendID,
		Name:                   "iRacing",
		SupportedImportFormats: pq.StringArray{"result-json", "telemetry-csv"},
		IsActive:               true,
	}

	svc := New()
	msg := svc.RacingSimToSimulation(input)

	if msg == nil {
		t.Fatal("expected converted message, got nil")
	}
	if msg.Id != 42 {
		t.Fatalf("unexpected id: got %d want %d", msg.Id, 42)
	}
	if msg.Name != "iRacing" {
		t.Fatalf("unexpected name: got %q want %q", msg.Name, "iRacing")
	}
	if !msg.IsActive {
		t.Fatal("unexpected active flag: got false want true")
	}
	if len(msg.SupportedFormats) != 2 {
		t.Fatalf("unexpected supported formats length: got %d want %d",
			len(msg.SupportedFormats), 2)
	}
	if msg.SupportedFormats[0] != "result-json" ||
		msg.SupportedFormats[1] != "telemetry-csv" {

		t.Fatalf("unexpected supported formats: got %v", msg.SupportedFormats)
	}

	input.SupportedImportFormats[0] = "changed"
	if msg.SupportedFormats[0] != "result-json" {
		t.Fatalf("supported formats should be copied, got %q", msg.SupportedFormats[0])
	}
}

func TestServiceRacingSimToSimulationNil(t *testing.T) {
	t.Parallel()

	svc := New()
	if msg := svc.RacingSimToSimulation(nil); msg != nil {
		t.Fatal("expected nil result for nil input")
	}
}

func TestServiceRacingSimsToSimulations(t *testing.T) {
	t.Parallel()

	svc := New()

	items := []*models.RacingSim{
		{
			ID:                     1,
			Name:                   "Assetto Corsa",
			SupportedImportFormats: pq.StringArray{"json"},
			IsActive:               true,
		},
		nil,
		{ID: 2, Name: "rFactor 2", SupportedImportFormats: pq.StringArray{"xml"}, IsActive: false},
	}

	got := svc.RacingSimsToSimulations(items)
	if len(got) != 2 {
		t.Fatalf("unexpected output length: got %d want %d", len(got), 2)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("unexpected ids: got %d, %d", got[0].Id, got[1].Id)
	}

	empty := svc.RacingSimsToSimulations(nil)
	if len(empty) != 0 {
		t.Fatalf("expected empty slice for nil input, got len=%d", len(empty))
	}
}
