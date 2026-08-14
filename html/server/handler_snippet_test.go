package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

//nolint:lll // test code
func TestParseSnippetRequest(t *testing.T) {
	r := httptest.NewRequestWithContext(
		context.Background(),
		"GET",
		"/snippet?view=standings&seasonID=12&eventID=34&classID=2&skipMode=never&cmsPath=https://cms.example.com",
		http.NoBody,
	)
	r.Header.Set("X-CMS-Base-Path", "https://header.example.com")

	req, err := newSnippetRequest(r)
	if err != nil {
		t.Fatalf("newSnippetRequest returned error: %v", err)
	}

	if req.View != "standings" {
		t.Fatalf("view = %q, want standings", req.View)
	}
	if req.SeasonID != 12 {
		t.Fatalf("seasonID = %d, want 12", req.SeasonID)
	}
	if req.EventID != 34 {
		t.Fatalf("eventID = %d, want 34", req.EventID)
	}
	if req.ClassID != 2 {
		t.Fatalf("classID = %d, want 2", req.ClassID)
	}
	if req.SkipMode != "never" {
		t.Fatalf("skipMode = %q, want never", req.SkipMode)
	}
	if req.CMSPath != "https://cms.example.com" {
		t.Fatalf("cmsPath = %q, want https://cms.example.com", req.CMSPath)
	}
}
