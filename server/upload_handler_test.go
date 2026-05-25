//nolint:lll,noctx,govet,funlen // test code, by design
package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
)

func TestParseRaceGridID(t *testing.T) {
	req := httptest.NewRequest("POST", "/upload/123", http.NoBody)
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
		{
			name:      "json",
			mediaType: "application/json",
			want:      commonv1.ImportFormat_IMPORT_FORMAT_JSON,
		},
		{
			name:      "json with suffix",
			mediaType: "application/vnd.api+json",
			want:      commonv1.ImportFormat_IMPORT_FORMAT_JSON,
		},
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

	uploads, err := parseMultipartUploadRequest(req)
	if err != nil {
		t.Fatalf("parseMultipartUploadRequest returned error: %v", err)
	}
	if len(uploads) != 1 {
		t.Fatalf("unexpected upload count: got %d want 1", len(uploads))
	}
	if uploads[0].format != commonv1.ImportFormat_IMPORT_FORMAT_JSON {
		t.Fatalf(
			"unexpected format: got %v want %v",
			uploads[0].format,
			commonv1.ImportFormat_IMPORT_FORMAT_JSON,
		)
	}
	if string(uploads[0].payload) != "{\"ok\":true}" {
		t.Fatalf("unexpected payload: %q", uploads[0].payload)
	}
}

func TestParseMultipartUploadRequest_MultipleParts(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	jsonPart, err := writer.CreatePart(textProtoMIMEHeader(
		"form-data; name=\"file\"; filename=\"results.json\"",
		"application/json",
	))
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := jsonPart.Write([]byte("{\"ok\":true}")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	xmlPart, err := writer.CreatePart(textProtoMIMEHeader(
		"form-data; name=\"payload\"; filename=\"results.xml\"",
		"application/xml",
	))
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := xmlPart.Write([]byte("<ok>true</ok>")); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	req := httptest.NewRequest("POST", "/upload/1", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	uploads, err := parseMultipartUploadRequest(req)
	if err != nil {
		t.Fatalf("parseMultipartUploadRequest returned error: %v", err)
	}
	if len(uploads) != 2 {
		t.Fatalf("unexpected upload count: got %d want 2", len(uploads))
	}
	if uploads[0].format != commonv1.ImportFormat_IMPORT_FORMAT_JSON {
		t.Fatalf(
			"unexpected first format: got %v want %v",
			uploads[0].format,
			commonv1.ImportFormat_IMPORT_FORMAT_JSON,
		)
	}
	if string(uploads[0].payload) != "{\"ok\":true}" {
		t.Fatalf("unexpected first payload: %q", uploads[0].payload)
	}
	if uploads[1].format != commonv1.ImportFormat_IMPORT_FORMAT_XML {
		t.Fatalf(
			"unexpected second format: got %v want %v",
			uploads[1].format,
			commonv1.ImportFormat_IMPORT_FORMAT_XML,
		)
	}
	if string(uploads[1].payload) != "<ok>true</ok>" {
		t.Fatalf("unexpected second payload: %q", uploads[1].payload)
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

	_, err = parseMultipartUploadRequest(req)
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
