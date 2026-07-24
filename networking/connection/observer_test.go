package connection

import (
	"context"
	"errors"
	"testing"

	"github.com/niflaot/pixels/networking/codec"
)

// TestObserverRegistryReceivesSuccessfulBidirectionalTraffic verifies tap timing.
func TestObserverRegistryReceivesSuccessfulBidirectionalTraffic(t *testing.T) {
	observers := NewObserverRegistry()
	directions := make([]Direction, 0, 2)
	if err := observers.Register(PacketObserverFunc(func(context Context, _ codec.Packet) {
		directions = append(directions, context.Direction)
	})); err != nil {
		t.Fatalf("register observer: %v", err)
	}
	fixture := sessionFixture(t)
	fixture.Observers = observers
	session := mustSession(t, fixture)
	if err := session.Receive(context.Background(), codec.Packet{Header: 1}); err != nil {
		t.Fatalf("receive: %v", err)
	}
	if err := session.Send(context.Background(), codec.Packet{Header: 2}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(directions) != 2 || directions[0] != InboundDirection || directions[1] != OutboundDirection {
		t.Fatalf("unexpected directions: %v", directions)
	}
}

// TestObserverRegistrySkipsFailedOutboundTraffic verifies only written packets are tapped.
func TestObserverRegistrySkipsFailedOutboundTraffic(t *testing.T) {
	observers := NewObserverRegistry()
	observed := 0
	if err := observers.Register(PacketObserverFunc(func(Context, codec.Packet) { observed++ })); err != nil {
		t.Fatalf("register observer: %v", err)
	}
	fixture := sessionFixture(t)
	fixture.Observers = observers
	fixture.Sender = func(context.Context, codec.Packet) error { return errors.New("write failed") }
	session := mustSession(t, fixture)
	if err := session.Send(context.Background(), codec.Packet{Header: 2}); err == nil {
		t.Fatal("expected send failure")
	}
	if observed != 0 {
		t.Fatalf("expected no observation, got %d", observed)
	}
}

// TestObserverRegistryRejectsNil verifies invalid observers are rejected.
func TestObserverRegistryRejectsNil(t *testing.T) {
	registry := NewObserverRegistry()
	if err := registry.Register(nil); !errors.Is(err, ErrInvalidHandler) {
		t.Fatalf("expected invalid handler, got %v", err)
	}
	if registry.Len() != 0 {
		t.Fatalf("expected empty registry, got %d", registry.Len())
	}
}
