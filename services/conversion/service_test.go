//nolint:lll // service tests can have long lines for test data setup
package conversion

import (
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/gofrs/uuid/v5"
	"github.com/lib/pq"

	"github.com/srlmgr/backend/db/models"
)

func TestImportFormatsToProto(t *testing.T) {
	t.Parallel()

	got := ImportFormatsToProto([]string{"json", "csv", "unknown"})
	if len(got) != 3 {
		t.Fatalf("unexpected format count: got %d want %d", len(got), 3)
	}
	if got[0] != commonv1.ImportFormat_IMPORT_FORMAT_JSON {
		t.Fatalf(
			"unexpected first format: got %v want %v",
			got[0],
			commonv1.ImportFormat_IMPORT_FORMAT_JSON,
		)
	}
	if got[1] != commonv1.ImportFormat_IMPORT_FORMAT_CSV {
		t.Fatalf(
			"unexpected second format: got %v want %v",
			got[1],
			commonv1.ImportFormat_IMPORT_FORMAT_CSV,
		)
	}
	if got[2] != commonv1.ImportFormat_IMPORT_FORMAT_UNSPECIFIED {
		t.Fatalf(
			"unexpected third format: got %v want %v",
			got[2],
			commonv1.ImportFormat_IMPORT_FORMAT_UNSPECIFIED,
		)
	}
}

func TestImportFormatsFromProto(t *testing.T) {
	t.Parallel()

	got, err := ImportFormatsFromProto([]commonv1.ImportFormat{
		commonv1.ImportFormat_IMPORT_FORMAT_JSON,
		commonv1.ImportFormat_IMPORT_FORMAT_CSV,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected format count: got %d want %d", len(got), 2)
	}
	if got[0] != "json" || got[1] != "csv" {
		t.Fatalf("unexpected formats: got %v", got)
	}
}

func TestImportFormatsFromProtoUnspecified(t *testing.T) {
	t.Parallel()

	got, err := ImportFormatsFromProto([]commonv1.ImportFormat{
		commonv1.ImportFormat_IMPORT_FORMAT_UNSPECIFIED,
	})
	if err != nil {
		t.Fatal("unexpected error for unspecified import format")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result for unspecified format, got %v", got)
	}
}

func TestServiceRacingSimToSimulation(t *testing.T) {
	t.Parallel()

	frontendID := uuid.Must(uuid.NewV4())
	input := &models.RacingSim{
		ID:                     42,
		FrontendID:             frontendID,
		Name:                   "iRacing",
		SupportedImportFormats: pq.StringArray{"json", "csv"},
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
	if msg.SupportedFormats[0] != commonv1.ImportFormat_IMPORT_FORMAT_JSON ||
		msg.SupportedFormats[1] != commonv1.ImportFormat_IMPORT_FORMAT_CSV {

		t.Fatalf("unexpected supported formats: got %v", msg.SupportedFormats)
	}

	input.SupportedImportFormats[0] = "csv"
	if msg.SupportedFormats[0] != commonv1.ImportFormat_IMPORT_FORMAT_JSON {
		t.Fatalf("supported formats should be copied, got %v", msg.SupportedFormats[0])
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
		{ID: 2, Name: "rFactor 2", SupportedImportFormats: pq.StringArray{"csv"}, IsActive: false},
	}

	got := svc.RacingSimsToSimulations(items)
	if len(got) != 2 {
		t.Fatalf("unexpected output length: got %d want %d", len(got), 2)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("unexpected ids: got %d, %d", got[0].Id, got[1].Id)
	}
	if got[0].SupportedFormats[0] != commonv1.ImportFormat_IMPORT_FORMAT_JSON {
		t.Fatalf("unexpected first item format: got %v", got[0].SupportedFormats[0])
	}
	if got[1].SupportedFormats[0] != commonv1.ImportFormat_IMPORT_FORMAT_CSV {
		t.Fatalf("unexpected second item format: got %v", got[1].SupportedFormats[0])
	}

	empty := svc.RacingSimsToSimulations(nil)
	if len(empty) != 0 {
		t.Fatalf("expected empty slice for nil input, got len=%d", len(empty))
	}
}
