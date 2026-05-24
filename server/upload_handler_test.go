package server

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
)

func TestParseRaceGridID(t *testing.T) {
	req := httptest.NewRequest("POST", "/upload/123", nil)
	req.SetPathValue("raceGridId", "123")

	got, err := parseRaceGridID(req)
	if err != nil {
		t.Fatalf("parseRaceGridID returned error: %v", err)
	}
	if got != 123 {
		t.Fatalf("unexpected raceGridID: got %d want 123", got)
	}
}

func TestDetectImportFormatFromMediaType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		want      commonv1.ImportFormat
		wantErr   bool
	}{
		{name: "json", mediaType: "application/json", want: commonv1.ImportFormat_IMPORT_FORMAT_JSON},
		{name: "json with suffix", mediaType: "application/vnd.api+json", want: commonv1.ImportFormat_IMPORT_FORMAT_JSON},
		{name: "csv", mediaType: "text/csv", want: commonv1.ImportFormat_IMPORT_FORMAT_CSV},
		{name: "xml", mediaType: "application/xml", want: commonv1.ImportFormat_IMPORT_FORMAT_XML},
		{name: "unsupported", mediaType: "application/octet-stream", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectImportFormatFromMediaType(tt.mediaType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for media type %q", tt.mediaType)
				}
				return
			}

			if err != nil {
				t.Fatalf("detectImportFormatFromMediaType returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected import format: got %v want %v", got, tt.want)
			}
		})
	}
}

func TestParseMultipartUploadRequest(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	partHeader := textProtoMIMEHeader(
		"form-data; name=\"file\"; filename=\"results.json\"",
		"application/json",
	)
	fileWriter, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := fileWriter.Write([]byte("{\"ok\":true}")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	req := httptest.NewRequest("POST", "/upload/1", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	gotFormat, payload, err := parseMultipartUploadRequest(req)
	if err != nil {
		t.Fatalf("parseMultipartUploadRequest returned error: %v", err)
	}
	if gotFormat != commonv1.ImportFormat_IMPORT_FORMAT_JSON {
		t.Fatalf("unexpected format: got %v want %v", gotFormat, commonv1.ImportFormat_IMPORT_FORMAT_JSON)
	}
	if string(payload) != "{\"ok\":true}" {
		t.Fatalf("unexpected payload: %q", payload)
	}
}

func TestParseMultipartUploadRequest_MissingFileContentType(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	partHeader := textProtoMIMEHeader(
		"form-data; name=\"file\"; filename=\"results.json\"",
		"",
	)
	fileWriter, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := fileWriter.Write([]byte("{\"ok\":true}")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	req := httptest.NewRequest("POST", "/upload/1", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, _, err = parseMultipartUploadRequest(req)
	if err == nil {
		t.Fatal("expected error for missing file content type")
	}
	if !strings.Contains(err.Error(), "file content type is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func textProtoMIMEHeader(contentDisposition, contentType string) textproto.MIMEHeader {
	header := textproto.MIMEHeader{
		"Content-Disposition": {contentDisposition},
	}
	if strings.TrimSpace(contentType) != "" {
		header.Set("Content-Type", contentType)
	}
	return header
}
