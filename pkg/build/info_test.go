package build

import "testing"

// TestDefaultInfo verifies the local development build metadata.
func TestDefaultInfo(t *testing.T) {
	info := DefaultInfo()

	if info.Name != Name {
		t.Fatalf("expected name %q, got %q", Name, info.Name)
	}

	if info.Version != Version+"-"+CommitHash {
		t.Fatalf("expected default version, got %q", info.Version)
	}
}

// TestNewInfoCombinesVersionAndCommit verifies build version formatting.
func TestNewInfoCombinesVersionAndCommit(t *testing.T) {
	info := NewInfo("pixels", "1.2.3", "1234567890abcdef")

	if info.Version != "1.2.3-12345678" {
		t.Fatalf("expected combined version, got %q", info.Version)
	}

	if info.Commit != "12345678" {
		t.Fatalf("expected short commit, got %q", info.Commit)
	}
}

// TestShortCommitKeepsShortValues verifies short commit values are stable.
func TestShortCommitKeepsShortValues(t *testing.T) {
	commit := ShortCommit("dev")

	if commit != "dev" {
		t.Fatalf("expected dev commit, got %q", commit)
	}
}
