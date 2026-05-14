package db

import (
	"encoding/json"
	"testing"
)

func TestExtractHeadersPreview_CapAtThree(t *testing.T) {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer xyz",
		"X-Custom-1":    "a",
		"X-Custom-2":    "b",
	}
	raw, _ := json.Marshal(headers)
	preview := extractHeadersPreview(raw)
	if len(preview) != 3 {
		t.Errorf("expected 3 headers, got %d", len(preview))
	}
	if preview["Content-Type"] != "application/json" {
		t.Error("expected Content-Type to always be included")
	}
}

func TestExtractHeadersPreview_FewerThanThree(t *testing.T) {
	headers := map[string]string{
		"Content-Type": "text/plain",
	}
	raw, _ := json.Marshal(headers)
	preview := extractHeadersPreview(raw)
	if len(preview) != 1 {
		t.Errorf("expected 1 header, got %d", len(preview))
	}
	if preview["Content-Type"] != "text/plain" {
		t.Errorf("expected Content-Type=text/plain, got %v", preview["Content-Type"])
	}
}

func TestExtractHeadersPreview_InvalidJSON(t *testing.T) {
	preview := extractHeadersPreview([]byte(`not-json`))
	if len(preview) != 0 {
		t.Errorf("expected empty map for invalid JSON, got %v", preview)
	}
}

func TestExtractHeadersPreview_Empty(t *testing.T) {
	preview := extractHeadersPreview([]byte(`{}`))
	if len(preview) != 0 {
		t.Errorf("expected empty map for empty headers, got %v", preview)
	}
}
