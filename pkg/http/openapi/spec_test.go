package openapi

import (
	"encoding/json"
	"testing"
)

// TestSpecIsJSON verifies the OpenAPI document is valid JSON.
func TestSpecIsJSON(t *testing.T) {
	var document map[string]any

	if err := json.Unmarshal(Bytes(), &document); err != nil {
		t.Fatalf("unmarshal spec: %v", err)
	}

	if document["openapi"] != "3.1.0" {
		t.Fatalf("expected OpenAPI 3.1.0, got %v", document["openapi"])
	}
}

// TestSpecDocumentsRoutes verifies the expected public routes are documented.
func TestSpecDocumentsRoutes(t *testing.T) {
	var document struct {
		Paths map[string]any `json:"paths"`
	}

	if err := json.Unmarshal(Bytes(), &document); err != nil {
		t.Fatalf("unmarshal spec: %v", err)
	}

	for _, path := range []string{"/status", "/ws", "/docs", "/*"} {
		if _, ok := document.Paths[path]; !ok {
			t.Fatalf("expected path %s to be documented", path)
		}
	}
}
