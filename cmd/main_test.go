package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewAppBuilds verifies the dependency graph can be constructed.
func TestNewAppBuilds(t *testing.T) {
	setI18NPathForTest(t)
	app := newApp()

	if app == nil {
		t.Fatal("expected app")
	}
}

// TestOptionsBuilds verifies dependency graph options are registered.
func TestOptionsBuilds(t *testing.T) {
	options := options()

	if len(options) != 18 {
		t.Fatalf("expected eighteen options, got %d", len(options))
	}
}

// setI18NPathForTest points app construction at an empty test catalog.
func setI18NPathForTest(t *testing.T) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "translations.json")
	if err := os.WriteFile(path, []byte(`{"locales":{}}`), 0o600); err != nil {
		t.Fatalf("write i18n catalog: %v", err)
	}
	t.Setenv("PIXELS_I18N_PATH", path)
}
