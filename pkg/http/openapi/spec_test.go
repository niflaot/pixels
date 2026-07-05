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

	if document["openapi"] != "3.0.3" {
		t.Fatalf("expected OpenAPI 3.0.3, got %v", document["openapi"])
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

// TestSpecGroupsRoutes verifies Scalar route sections are documented.
func TestSpecGroupsRoutes(t *testing.T) {
	var document struct {
		Tags  []map[string]string           `json:"tags"`
		Paths map[string]map[string]opGroup `json:"paths"`
	}

	if err := json.Unmarshal(Bytes(), &document); err != nil {
		t.Fatalf("unmarshal spec: %v", err)
	}

	if !hasTag(document.Tags, "Admin Connections") {
		t.Fatal("expected admin connection tag")
	}

	groups := document.Paths["/api/admin/connections"]["get"].Tags
	if len(groups) != 1 || groups[0] != "Admin Connections" {
		t.Fatalf("expected admin route group, got %#v", groups)
	}
}

// opGroup contains the OpenAPI operation groups needed by tests.
type opGroup struct {
	// Tags stores the operation tag list.
	Tags []string `json:"tags"`
}

// hasTag reports whether the OpenAPI document contains a tag.
func hasTag(tags []map[string]string, name string) bool {
	for _, tag := range tags {
		if tag["name"] == name {
			return true
		}
	}

	return false
}
