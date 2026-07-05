package logger

import "testing"

// TestNewBuildsConsoleLogger verifies console logger construction.
func TestNewBuildsConsoleLogger(t *testing.T) {
	log, err := New(Config{
		Level:  "debug",
		Format: FormatConsole,
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	if log == nil {
		t.Fatal("expected logger")
	}
}

// TestNewBuildsJSONLogger verifies JSON logger construction.
func TestNewBuildsJSONLogger(t *testing.T) {
	log, err := New(Config{
		Level:  "info",
		Format: FormatJSON,
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	if log == nil {
		t.Fatal("expected logger")
	}
}

// TestNewRejectsInvalidLevel verifies invalid zap levels fail construction.
func TestNewRejectsInvalidLevel(t *testing.T) {
	_, err := New(Config{
		Level:  "verbose",
		Format: FormatConsole,
	})
	if err == nil {
		t.Fatal("expected invalid level error")
	}
}

// TestNewRejectsInvalidFormat verifies unsupported encoders fail construction.
func TestNewRejectsInvalidFormat(t *testing.T) {
	_, err := New(Config{
		Level:  "info",
		Format: "text",
	})
	if err == nil {
		t.Fatal("expected invalid format error")
	}
}
