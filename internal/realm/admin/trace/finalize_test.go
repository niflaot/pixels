package trace

import (
	"context"
	"errors"
	"testing"

	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
)

// TestFinalizeIsIdempotent verifies repeated calls upload only once.
func TestFinalizeIsIdempotent(t *testing.T) {
	fixture := newTraceFixture(t)
	if _, _, err := fixture.tracer.Activate(context.Background(), 7, "demo"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	fixture.tracer.capture(7, netconn.Context{ConnectionID: "one", Direction: netconn.InboundDirection}, codec.Packet{Header: 1})
	if _, err := fixture.tracer.Finalize(context.Background(), 7, "manual"); err != nil {
		t.Fatalf("first finalize: %v", err)
	}
	if _, err := fixture.tracer.Finalize(context.Background(), 7, "manual"); !errors.Is(err, ErrInactive) {
		t.Fatalf("expected inactive second finalize, got %v", err)
	}
	calls, key, _ := fixture.uploader.snapshot()
	if calls != 1 || key != "debug/traces/trace-id.txt" {
		t.Fatalf("calls=%d key=%q", calls, key)
	}
}

// TestFinalizeFailureRemainsRetryable verifies upload failures preserve trace state.
func TestFinalizeFailureRemainsRetryable(t *testing.T) {
	fixture := newTraceFixture(t)
	fixture.uploader.err = errors.New("storage unavailable")
	if _, _, err := fixture.tracer.Activate(context.Background(), 7, "demo"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := fixture.tracer.Finalize(context.Background(), 7, "manual"); err == nil {
		t.Fatal("expected upload failure")
	}
	if session, active := fixture.tracer.Active(7); !active || session.finalizing {
		t.Fatalf("expected retryable active session: %+v", session)
	}
	fixture.uploader.err = nil
	if _, err := fixture.tracer.Finalize(context.Background(), 7, "manual"); err != nil {
		t.Fatalf("retry finalize: %v", err)
	}
}
