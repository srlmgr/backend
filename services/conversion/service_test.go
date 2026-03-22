//nolint:lll,funlen // service tests can have long lines for test data setup
package conversion

import (
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/gofrs/uuid/v5"
	"github.com/lib/pq"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/srlmgr/backend/db/models"
)

func TestImportFormatsToProto(t *testing.T) {
	t.Parallel()

	got := ImportFormatsToProto([]string{ImportFormatJSON, ImportFormatCSV, "unknown"})
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
	if got[0] != ImportFormatJSON || got[1] != ImportFormatCSV {
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
		SupportedImportFormats: pq.StringArray{ImportFormatJSON, ImportFormatCSV},
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

	input.SupportedImportFormats[0] = ImportFormatCSV
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
			SupportedImportFormats: pq.StringArray{ImportFormatJSON},
			IsActive:               true,
		},
		nil,
		{
			ID:                     2,
			Name:                   "rFactor 2",
			SupportedImportFormats: pq.StringArray{ImportFormatCSV},
			IsActive:               false,
		},
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

func TestServiceEventToEvent(t *testing.T) {
	t.Parallel()

	eventDate := timestamppb.Now().AsTime()
	input := &models.Event{
		ID:              42,
		SeasonID:        3,
		TrackLayoutID:   5,
		Name:            "Round 1",
		EventDate:       eventDate,
		Status:          EventStatusScheduled,
		ProcessingState: EventProcessingStateRawImported,
	}

	svc := New()
	msg := svc.EventToEvent(input)

	if msg == nil {
		t.Fatal("expected converted event, got nil")
	}
	if msg.GetId() != 42 {
		t.Fatalf("unexpected id: got %d want %d", msg.GetId(), 42)
	}
	if msg.GetSeasonId() != 3 {
		t.Fatalf("unexpected season_id: got %d want %d", msg.GetSeasonId(), 3)
	}
	if msg.GetTrackLayoutId() != 5 {
		t.Fatalf("unexpected track_layout_id: got %d want %d", msg.GetTrackLayoutId(), 5)
	}
	if msg.GetName() != "Round 1" {
		t.Fatalf("unexpected name: got %q want %q", msg.GetName(), "Round 1")
	}
	if !msg.GetEventDate().AsTime().Equal(eventDate) {
		t.Fatalf("unexpected event_date: got %v want %v", msg.GetEventDate().AsTime(), eventDate)
	}
	if msg.GetStatus() != commonv1.EventStatus_EVENT_STATUS_SCHEDULED {
		t.Fatalf(
			"unexpected status: got %v want %v",
			msg.GetStatus(),
			commonv1.EventStatus_EVENT_STATUS_SCHEDULED,
		)
	}
	if msg.GetProcessingState() != commonv1.EventProcessingState_EVENT_PROCESSING_STATE_RAW_IMPORTED {
		t.Fatalf(
			"unexpected processing_state: got %v want %v",
			msg.GetProcessingState(),
			commonv1.EventProcessingState_EVENT_PROCESSING_STATE_RAW_IMPORTED,
		)
	}

	input.Status = "unknown_status"
	input.ProcessingState = "unknown_processing_state"
	msg = svc.EventToEvent(input)
	if msg.GetStatus() != commonv1.EventStatus_EVENT_STATUS_UNSPECIFIED {
		t.Fatalf(
			"expected unknown status to fall back to zero value, got %v",
			msg.GetStatus(),
		)
	}
	if msg.GetProcessingState() != commonv1.EventProcessingState_EVENT_PROCESSING_STATE_UNSPECIFIED {
		t.Fatalf(
			"expected unknown processing_state to fall back to zero value, got %v",
			msg.GetProcessingState(),
		)
	}
}
