package logger

import (
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
)

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

// TestNewBuildsToonConsoleLogger verifies compact toon logger output.
func TestNewBuildsToonConsoleLogger(t *testing.T) {
	path := t.TempDir() + "/toon.log"
	config, err := buildConfig(Config{
		Level:       "debug",
		Format:      FormatJSON,
		ToonConsole: true,
	})
	if err != nil {
		t.Fatalf("build config: %v", err)
	}

	config.OutputPaths = []string{path}
	log, err := config.Build()
	if err != nil {
		t.Fatalf("build logger: %v", err)
	}

	log.Debug("toon packet",
		zap.String("connection_id", "12345678-1234-1234-1234-123456789012"),
		zap.String("connection_kind", "websocket"),
		zap.Uint16("packet_header", 4000),
	)
	if err := log.Sync(); err != nil {
		t.Fatalf("sync logger: %v", err)
	}

	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read logger output: %v", err)
	}

	if strings.Contains(string(output), "caller") || strings.Contains(string(output), "ts:") {
		t.Fatalf("expected compact output without timestamp/caller, got %q", output)
	}
	if strings.Count(strings.TrimSpace(string(output)), "\n") != 0 {
		t.Fatalf("expected one inline entry, got %q", output)
	}
	if !strings.Contains(string(output), "lvl: 0, msg: toon packet") {
		t.Fatalf("expected numeric debug level, got %q", output)
	}
	if !strings.Contains(string(output), ", cid: \"12345678\"") || !strings.Contains(string(output), ", header: 4000") {
		t.Fatalf("expected compact toon fields, got %q", output)
	}
	if strings.Contains(string(output), "connection_kind") {
		t.Fatalf("expected connection kind to be removed, got %q", output)
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

// TestFormatToonLinePreservesTraceOrder verifies exported trace formatting.
func TestFormatToonLinePreservesTraceOrder(t *testing.T) {
	line := FormatToonLine(map[string]any{
		"payload": "AQI=",
		"bytes":   2,
		"header":  uint16(4000),
		"cid":     "12345678",
		"dir":     "in",
		"ts":      "2026-07-24T02:15:03.412Z",
		"seq":     1,
	})
	expected := "seq: 1, ts: \"2026-07-24T02:15:03.412Z\", dir: in, cid: \"12345678\", header: 4000, bytes: 2, payload: AQI=\n"
	if line != expected {
		t.Fatalf("expected %q, got %q", expected, line)
	}
}
