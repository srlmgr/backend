package bookings

import (
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"

	"github.com/srlmgr/backend/db/models"
	mytypes "github.com/srlmgr/backend/db/mytypes"
)

func TestBookingEntryToProtoMapsOfftracksExceededSource(t *testing.T) {
	t.Parallel()

	entry := bookingEntryToProto(&models.BookingEntry{
		SourceType: mytypes.SourceType("offtracks_exceeded"),
	})
	got := entry.GetSourceType()
	want := commonv1.BookingSourceType_BOOKING_SOURCE_TYPE_OFFTRACKS_EXCEEDED
	if got != want {
		t.Fatalf("source type = %v, want %v", got, want)
	}
}
