package build

import "testing"

// TestDefaultInfo verifies the local development build metadata.
func TestDefaultInfo(t *testing.T) {
	info := DefaultInfo()

	if info.Name != "pixels" {
		t.Fatalf("expected name pixels, got %q", info.Name)
	}

	if info.Version != "dev" {
		t.Fatalf("expected version dev, got %q", info.Version)
	}
}
