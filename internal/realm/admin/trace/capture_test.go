package trace

import (
	"context"
	"testing"

	"github.com/niflaot/pixels/internal/realm/session/binding"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
)

// TestObserveCapturesBothDirectionsAcrossReconnect verifies player-indexed tracing.
func TestObserveCapturesBothDirectionsAcrossReconnect(t *testing.T) {
	fixture := newTraceFixture(t)
	if _, _, err := fixture.tracer.Activate(context.Background(), 7, "demo"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	bindTracePlayer(t, fixture.bindings, 7, "connection-one")
	fixture.tracer.Observe(netconn.Context{ConnectionID: "connection-one", ConnectionKind: "websocket", Direction: netconn.InboundDirection}, codec.Packet{Header: 100, Payload: []byte{1, 2}})
	fixture.bindings.RemoveByPlayer(7)
	bindTracePlayer(t, fixture.bindings, 7, "connection-two")
	fixture.tracer.Observe(netconn.Context{ConnectionID: "connection-two", ConnectionKind: "websocket", Direction: netconn.OutboundDirection}, codec.Packet{Header: 200, Payload: []byte{3}})

	result, err := fixture.tracer.Finalize(context.Background(), 7, "manual")
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	_, _, body := fixture.uploader.snapshot()
	if result.Count != 2 || !containsAll(body, "dir: in", "header: 100", "dir: out", "header: 200", "AQI=", "Aw==") {
		t.Fatalf("result=%+v body=%q", result, body)
	}
}

// TestCaptureStopsAtSafetyCap verifies truncation protects persistence.
func TestCaptureStopsAtSafetyCap(t *testing.T) {
	fixture := newTraceFixture(t)
	fixture.tracer.maxEntries = 1
	if _, _, err := fixture.tracer.Activate(context.Background(), 7, "demo"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	connection := netconn.Context{ConnectionID: "one", ConnectionKind: "websocket", Direction: netconn.InboundDirection}
	fixture.tracer.capture(7, connection, codec.Packet{Header: 1})
	fixture.tracer.capture(7, connection, codec.Packet{Header: 2})

	result, err := fixture.tracer.Finalize(context.Background(), 7, "manual")
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	_, _, body := fixture.uploader.snapshot()
	if result.Count != 1 || !result.Truncated || !containsAll(body, "truncated: true", "-- truncado:") {
		t.Fatalf("result=%+v body=%q", result, body)
	}
}

// BenchmarkObserveInactive measures the Redis-free inactive traffic path.
func BenchmarkObserveInactive(b *testing.B) {
	tracer := &Tracer{bindings: binding.NewRegistry(), sessions: make(map[int64]*Session)}
	_ = tracer.bindings.Add(binding.Binding{PlayerID: 7, ConnectionID: "one", ConnectionKind: "websocket"})
	connection := netconn.Context{ConnectionID: "one", ConnectionKind: "websocket", Direction: netconn.InboundDirection}
	packet := codec.Packet{Header: 1}
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		tracer.Observe(connection, packet)
	}
}
