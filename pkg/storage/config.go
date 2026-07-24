// Package storage provides reusable S3-compatible object storage.
package storage

import (
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config contains S3-compatible object storage settings.
type Config struct {
	// Endpoint stores the host and optional port without a URL scheme.
	Endpoint string `env:"STORAGE_ENDPOINT" envDefault:"127.0.0.1:9000"`
	// PublicBaseURL overrides the durable public bucket URL.
	PublicBaseURL string `env:"STORAGE_PUBLIC_BASE_URL" envDefault:""`
	// AccessKey stores the S3 access key.
	AccessKey string `env:"STORAGE_ACCESS_KEY" envDefault:""`
	// SecretKey stores the S3 secret key.
	SecretKey string `env:"STORAGE_SECRET_KEY" envDefault:""`
	// Bucket stores the durable object bucket.
	Bucket string `env:"STORAGE_BUCKET" envDefault:"pixels-camera"`
	// UseSSL enables HTTPS for S3 operations.
	UseSSL bool `env:"STORAGE_USE_SSL" envDefault:"true"`
	// PublicRead applies an idempotent public read-only bucket policy.
	PublicRead bool `env:"STORAGE_PUBLIC_READ" envDefault:"true"`
	// UploadTimeout bounds each upload and delete request.
	UploadTimeout time.Duration `env:"STORAGE_UPLOAD_TIMEOUT" envDefault:"10s"`
}

// DebugConfig contains the bucket-specific settings for diagnostic objects.
type DebugConfig struct {
	// PublicBaseURL overrides the durable public debug bucket URL.
	PublicBaseURL string `env:"STORAGE_DEBUG_PUBLIC_BASE_URL" envDefault:""`
	// Bucket stores packet traces and future diagnostic objects.
	Bucket string `env:"STORAGE_DEBUG_BUCKET" envDefault:"pixels-debug"`
}

// LoadConfig reads storage configuration from environment variables.
func LoadConfig() (Config, error) { return env.ParseAs[Config]() }

// LoadDebugConfig reads diagnostic storage configuration from environment variables.
func LoadDebugConfig() (DebugConfig, error) { return env.ParseAs[DebugConfig]() }

// valid reports whether required fields and limits are usable.
func (config Config) valid() bool {
	return strings.TrimSpace(config.Endpoint) != "" && strings.TrimSpace(config.Bucket) != "" && config.UploadTimeout > 0
}

// apply returns shared storage settings scoped to the diagnostic bucket.
func (config DebugConfig) apply(shared Config) Config {
	shared.PublicBaseURL = config.PublicBaseURL
	shared.Bucket = config.Bucket
	return shared
}
