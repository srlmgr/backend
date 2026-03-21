//nolint:lll,funlen // test setup
package conversion

import (
	"testing"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestProtoFieldStringOrEnumValueEnumField(t *testing.T) {
	t.Parallel()

	msg, statusField, processingStateField := newDynamicEventMessage(t)
	msg.Set(statusField, protoreflect.ValueOfEnum(1))
	msg.Set(processingStateField, protoreflect.ValueOfEnum(2))

	status, err := ProtoFieldStringOrEnumValue(msg, "status")
	if err != nil {
		t.Fatalf("unexpected status error: %v", err)
	}
	if status != "scheduled" {
		t.Fatalf("unexpected status: got %q want %q", status, "scheduled")
	}

	processingState, err := ProtoFieldStringOrEnumValue(msg, "processing_state")
	if err != nil {
		t.Fatalf("unexpected processing state error: %v", err)
	}
	if processingState != "raw_imported" {
		t.Fatalf("unexpected processing_state: got %q want %q", processingState, "raw_imported")
	}
}

func TestSetProtoFieldStringOrEnumEnumField(t *testing.T) {
	t.Parallel()

	msg, statusField, processingStateField := newDynamicEventMessage(t)

	if err := SetProtoFieldStringOrEnum(msg, "status", "completed"); err != nil {
		t.Fatalf("unexpected status error: %v", err)
	}
	if got := msg.Get(statusField).Enum(); got != 2 {
		t.Fatalf("unexpected status enum number: got %d want %d", got, 2)
	}

	if err := SetProtoFieldStringOrEnum(msg, "processing_state", "finalized"); err != nil {
		t.Fatalf("unexpected processing state error: %v", err)
	}
	if got := msg.Get(processingStateField).Enum(); got != 3 {
		t.Fatalf("unexpected processing_state enum number: got %d want %d", got, 3)
	}
}

func TestSetProtoFieldStringOrEnumUnknownValueFallsBackToUnspecified(t *testing.T) {
	t.Parallel()

	msg, statusField, _ := newDynamicEventMessage(t)

	err := SetProtoFieldStringOrEnum(msg, "status", "not_real")
	if err == nil {
		t.Fatal("expected error for unsupported persisted value")
	}
	if got := msg.Get(statusField).Enum(); got != 0 {
		t.Fatalf("unexpected fallback enum number: got %d want %d", got, 0)
	}
}

//nolint:whitespace // editor/linter issue
func newDynamicEventMessage(
	t *testing.T,
) (*dynamicpb.Message, protoreflect.FieldDescriptor, protoreflect.FieldDescriptor) {
	t.Helper()

	fileDescriptor := &descriptorpb.FileDescriptorProto{
		Syntax:  protoString("proto3"),
		Name:    protoString("test/event.proto"),
		Package: protoString("test"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: protoString("EventStatus"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: protoString("EVENT_STATUS_UNSPECIFIED"), Number: protoInt32(0)},
					{Name: protoString("EVENT_STATUS_SCHEDULED"), Number: protoInt32(1)},
					{Name: protoString("EVENT_STATUS_COMPLETED"), Number: protoInt32(2)},
				},
			},
			{
				Name: protoString("EventProcessingState"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{
						Name:   protoString("EVENT_PROCESSING_STATE_UNSPECIFIED"),
						Number: protoInt32(0),
					},
					{
						Name:   protoString("EVENT_PROCESSING_STATE_RAW_IMPORTED"),
						Number: protoInt32(2),
					},
					{Name: protoString("EVENT_PROCESSING_STATE_FINALIZED"), Number: protoInt32(3)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: protoString("EventLike"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     protoString("status"),
						Number:   protoInt32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: protoString(".test.EventStatus"),
					},
					{
						Name:     protoString("processing_state"),
						Number:   protoInt32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: protoString(".test.EventProcessingState"),
					},
				},
			},
		},
	}

	file, err := protodesc.NewFile(fileDescriptor, nil)
	if err != nil {
		t.Fatalf("failed to create dynamic file descriptor: %v", err)
	}

	messageDescriptor := file.Messages().ByName("EventLike")
	if messageDescriptor == nil {
		t.Fatal("expected EventLike message descriptor")
	}

	statusField := messageDescriptor.Fields().ByName("status")
	processingStateField := messageDescriptor.Fields().ByName("processing_state")
	return dynamicpb.NewMessage(messageDescriptor), statusField, processingStateField
}

func protoString(value string) *string {
	return &value
}

func protoInt32(value int32) *int32 {
	return &value
}
