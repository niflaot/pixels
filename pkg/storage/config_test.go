package storage

import (
	"testing"
	"time"
)

// TestLoadConfigDefaults verifies complete storage defaults.
func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("STORAGE_ENDPOINT", "")
	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Endpoint != "127.0.0.1:9000" || config.Bucket != "pixels-camera" || config.UploadTimeout != 10*time.Second || !config.UseSSL || !config.PublicRead {
		t.Fatalf("config=%+v", config)
	}
}

// TestLoadDebugConfigDefaults verifies diagnostic objects use an independent bucket.
func TestLoadDebugConfigDefaults(t *testing.T) {
	t.Setenv("STORAGE_DEBUG_BUCKET", "")
	t.Setenv("STORAGE_DEBUG_PUBLIC_BASE_URL", "")
	config, err := LoadDebugConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Bucket != "pixels-debug" || config.PublicBaseURL != "" {
		t.Fatalf("config=%+v", config)
	}
}

// TestDebugConfigApplyPreservesSharedCredentials verifies only bucket routing changes.
func TestDebugConfigApplyPreservesSharedCredentials(t *testing.T) {
	shared := Config{Endpoint: "storage.local", AccessKey: "access", SecretKey: "secret", Bucket: "pixels-camera", PublicBaseURL: "https://cdn.local/camera"}
	applied := (DebugConfig{Bucket: "pixels-debug", PublicBaseURL: "https://cdn.local/debug"}).apply(shared)
	if applied.Endpoint != shared.Endpoint || applied.AccessKey != shared.AccessKey || applied.SecretKey != shared.SecretKey {
		t.Fatalf("shared settings changed: %+v", applied)
	}
	if applied.Bucket != "pixels-debug" || applied.PublicBaseURL != "https://cdn.local/debug" {
		t.Fatalf("debug routing not applied: %+v", applied)
	}
}
