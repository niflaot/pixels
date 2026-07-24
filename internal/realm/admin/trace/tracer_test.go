package trace

import (
	"context"
	"testing"
	"time"

	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
)

// TestActivateIsIdempotentAndAbsolute verifies repeated activation keeps one deadline.
func TestActivateIsIdempotentAndAbsolute(t *testing.T) {
	fixture := newTraceFixture(t)
	now := time.Unix(1_000, 0).UTC()
	fixture.tracer.now = func() time.Time { return now }
	first, created, err := fixture.tracer.Activate(context.Background(), 7, "demo")
	if err != nil || !created {
		t.Fatalf("activate first created=%v err=%v", created, err)
	}
	now = now.Add(10 * time.Minute)
	second, created, err := fixture.tracer.Activate(context.Background(), 7, "demo")
	if err != nil || created {
		t.Fatalf("activate second created=%v err=%v", created, err)
	}
	if !first.ExpiresAt.Equal(second.ExpiresAt) || !second.ExpiresAt.Equal(first.StartedAt.Add(DefaultDuration)) {
		t.Fatalf("deadlines changed: first=%s second=%s", first.ExpiresAt, second.ExpiresAt)
	}
}

// TestReapUsesAbsoluteDeadline verifies expiry finalizes without extending activity.
func TestReapUsesAbsoluteDeadline(t *testing.T) {
	fixture := newTraceFixture(t)
	now := time.Unix(2_000, 0).UTC()
	fixture.tracer.now = func() time.Time { return now }
	session, _, err := fixture.tracer.Activate(context.Background(), 7, "demo")
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	now = session.ExpiresAt
	fixture.tracer.capture(7, netconn.Context{Direction: netconn.InboundDirection}, codec.Packet{Header: 1})
	fixture.tracer.reap(context.Background())
	if _, active := fixture.tracer.Active(7); active {
		t.Fatal("expected expired trace removed")
	}
	calls, _, body := fixture.uploader.snapshot()
	if calls != 1 || !containsAll(body, "reason: expired", "player: demo (7)", "packet_count: 0") {
		t.Fatalf("calls=%d body=%q", calls, body)
	}
}
