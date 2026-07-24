// Package trace captures bounded production packet traces for staff commands.
package trace

import (
	"context"
	"errors"
	"io"
	"time"
)

const (
	// DefaultDuration is the absolute packet trace window.
	DefaultDuration = 30 * time.Minute
	// DefaultMaxEntries is the maximum number of packets in one trace.
	DefaultMaxEntries = 20_000
	// DefaultMaxBytes is the maximum encoded trace size before truncation.
	DefaultMaxBytes = 20 * 1024 * 1024
	// redisRetention keeps interrupted traces available for reconciliation.
	redisRetention = 24 * time.Hour
	// writeQueueSize bounds pending Redis persistence work.
	writeQueueSize = 2_048
	// reaperInterval controls expiration checks.
	reaperInterval = 30 * time.Second
)

var (
	// ErrInactive reports a player without an active trace.
	ErrInactive = errors.New("packet trace inactive")
	// ErrFinalizing reports a trace already being finalized.
	ErrFinalizing = errors.New("packet trace finalizing")
)

// Session stores durable packet trace metadata.
type Session struct {
	// PlayerID identifies the traced player.
	PlayerID int64 `json:"player_id"`
	// PlayerName stores the player name at activation.
	PlayerName string `json:"player_name"`
	// StartedAt stores the fixed trace start.
	StartedAt time.Time `json:"started_at"`
	// ExpiresAt stores the absolute trace deadline.
	ExpiresAt time.Time `json:"expires_at"`
	// Count stores accepted packet entries.
	Count int `json:"count"`
	// Bytes stores encoded packet entry bytes.
	Bytes int64 `json:"bytes"`
	// Truncated reports whether a safety limit stopped capture.
	Truncated bool `json:"truncated"`
	// finalizing prevents duplicate uploads in one process.
	finalizing bool
}

// Result summarizes one finalized trace.
type Result struct {
	// URL is the durable uploaded trace location.
	URL string
	// Count is the number of persisted packet entries.
	Count int
	// Truncated reports whether capture hit a safety limit.
	Truncated bool
}

// Store contains Redis operations required by packet traces.
type Store interface {
	// Delete removes one Redis key.
	Delete(context.Context, string) error
	// Expire assigns a Redis key expiration.
	Expire(context.Context, string, time.Duration) error
	// Find reads one Redis string.
	Find(context.Context, string) ([]byte, bool, error)
	// ListAppend appends values to a Redis list.
	ListAppend(context.Context, string, ...[]byte) error
	// ListRange reads an inclusive Redis list range.
	ListRange(context.Context, string, int64, int64) ([][]byte, error)
	// Set writes one Redis string.
	Set(context.Context, string, []byte, time.Duration) error
	// SetAdd adds members to a Redis set.
	SetAdd(context.Context, string, ...string) error
	// SetIfAbsent acquires one expiring Redis lock.
	SetIfAbsent(context.Context, string, []byte, time.Duration) (bool, error)
	// SetMembers returns Redis set members.
	SetMembers(context.Context, string) ([]string, error)
	// SetRemove removes Redis set members.
	SetRemove(context.Context, string, ...string) error
}

// Uploader stores finalized trace documents.
type Uploader interface {
	// Put uploads one object and returns its public URL.
	Put(context.Context, string, io.Reader, int64, string) (string, error)
}

// writeRequest stores one ordered asynchronous persistence operation.
type writeRequest struct {
	// session stores the latest durable metadata.
	session Session
	// line stores an optional packet entry.
	line []byte
	// done receives barrier persistence results.
	done chan error
}
