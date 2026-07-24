package trace

import (
	"context"
	"testing"
	"time"

	realmconn "github.com/niflaot/pixels/internal/realm/connection"
	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	"github.com/niflaot/pixels/internal/realm/session/binding"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// traceLifecycle records lifecycle hooks.
type traceLifecycle struct {
	// hooks stores appended startup and shutdown hooks.
	hooks []fx.Hook
}

// Append records one lifecycle hook.
func (lifecycle *traceLifecycle) Append(hook fx.Hook) {
	lifecycle.hooks = append(lifecycle.hooks, hook)
}

// TestStartReconcilesInterruptedTrace verifies process restarts finalize Redis state.
func TestStartReconcilesInterruptedTrace(t *testing.T) {
	first := newTraceFixture(t)
	if _, _, err := first.tracer.Activate(context.Background(), 7, "demo"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	first.tracer.capture(7, netconn.Context{ConnectionID: "one", Direction: netconn.InboundDirection}, codec.Packet{Header: 99})
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	if err := first.tracer.Stop(stopCtx); err != nil {
		cancel()
		t.Fatalf("stop first tracer: %v", err)
	}
	cancel()

	recoveredUploads := &traceUploader{}
	second := newTraceFixtureWithStore(t, first.redis, recoveredUploads)
	if _, active := second.tracer.Active(7); active {
		t.Fatal("expected orphan finalized instead of restored")
	}
	calls, _, body := recoveredUploads.snapshot()
	if calls != 1 || !containsAll(body, "reason: server restarted", "header: 99") {
		t.Fatalf("calls=%d body=%q", calls, body)
	}
	members, err := first.redis.SetMembers(context.Background(), activeKey)
	if err != nil || len(members) != 0 {
		t.Fatalf("members=%q err=%v", members, err)
	}
}

// TestNewAndRegisterConfigureDefaults verifies production constructor wiring.
func TestNewAndRegisterConfigureDefaults(t *testing.T) {
	fixture := newTraceFixture(t)
	tracer := New(fixture.redis, nil, playerlive.NewRegistry(), binding.NewRegistry(), netconn.NewRegistry(), nil, zap.NewNop())
	if tracer.duration != DefaultDuration || tracer.maxEntries != DefaultMaxEntries || tracer.maxBytes != DefaultMaxBytes {
		t.Fatalf("unexpected defaults: duration=%s entries=%d bytes=%d", tracer.duration, tracer.maxEntries, tracer.maxBytes)
	}
	lifecycle := &traceLifecycle{}
	handlers := &realmconn.Handlers{Observers: netconn.NewObserverRegistry()}
	if err := Register(lifecycle, handlers, tracer); err != nil {
		t.Fatalf("register tracer: %v", err)
	}
	if len(lifecycle.hooks) != 1 || handlers.Observers.Len() != 1 {
		t.Fatalf("hooks=%d observers=%d", len(lifecycle.hooks), handlers.Observers.Len())
	}
}
