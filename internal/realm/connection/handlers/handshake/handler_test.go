package handshake

import (
	"context"
	"errors"
	"testing"

	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	"github.com/niflaot/pixels/networking/crypto/diffie"
	indiffiecomplete "github.com/niflaot/pixels/networking/inbound/handshake/diffie/complete"
	indiffieinit "github.com/niflaot/pixels/networking/inbound/handshake/diffie/init"
	inpolicy "github.com/niflaot/pixels/networking/inbound/handshake/policy"
	inrelease "github.com/niflaot/pixels/networking/inbound/handshake/release"
	invariables "github.com/niflaot/pixels/networking/inbound/handshake/variables"
	outdiffieinit "github.com/niflaot/pixels/networking/outbound/handshake/diffie/init"
)

// TestDiffieInitRejectsUnexpectedHeader verifies inbound header validation.
func TestDiffieInitRejectsUnexpectedHeader(t *testing.T) {
	err := (Handler{}).DiffieInit(netconn.Context{}, codec.Packet{Header: 1})
	if !errors.Is(err, codec.ErrUnexpectedHeader) {
		t.Fatalf("expected unexpected header, got %v", err)
	}
}

// TestRegisterHandlesEarlyHandshake verifies early metadata packets.
func TestRegisterHandlesEarlyHandshake(t *testing.T) {
	session := testSession(t)

	if err := session.Receive(context.Background(), releasePacket(t)); err != nil {
		t.Fatalf("receive release: %v", err)
	}
	if err := session.Receive(context.Background(), variablesPacket(t)); err != nil {
		t.Fatalf("receive variables: %v", err)
	}
	if err := session.Receive(context.Background(), policyPacket(t)); err != nil {
		t.Fatalf("receive policy: %v", err)
	}
}

// TestDiffieInitBeginsConfiguredNegotiation verifies signed parameters are sent.
func TestDiffieInitBeginsConfiguredNegotiation(t *testing.T) {
	factory := testDiffieFactory(t)
	var sent codec.Packet
	session := testSessionWithFactory(t, factory, func(packet codec.Packet) {
		sent = packet
	})

	if err := session.Receive(context.Background(), releasePacket(t)); err != nil {
		t.Fatalf("receive release: %v", err)
	}
	if err := session.Receive(context.Background(), diffieInitPacket(t)); err != nil {
		t.Fatalf("receive diffie init: %v", err)
	}
	if sent.Header != outdiffieinit.Header {
		t.Fatalf("expected Diffie init response, got %d with states %d and %d", sent.Header, session.State(), session.SecurityState())
	}
	if session.State() != netconn.StateSecuring || session.SecurityState() != netconn.SecurityNegotiating {
		t.Fatalf("unexpected negotiation states %d and %d", session.State(), session.SecurityState())
	}
}

// TestDiffieInitUnavailableDisconnects verifies placeholder Diffie failure.
func TestDiffieInitUnavailableDisconnects(t *testing.T) {
	session := testSession(t)

	if err := session.Receive(context.Background(), releasePacket(t)); err != nil {
		t.Fatalf("receive release: %v", err)
	}
	if err := session.Receive(context.Background(), diffieInitPacket(t)); err != nil {
		t.Fatalf("receive diffie init: %v", err)
	}
	if session.State() != netconn.StateClosed {
		t.Fatalf("expected closed session, got %d", session.State())
	}
}

// TestDiffieCompleteUnavailableDisconnects verifies completion failure.
func TestDiffieCompleteUnavailableDisconnects(t *testing.T) {
	session := testSession(t)

	if err := session.Receive(context.Background(), releasePacket(t)); err != nil {
		t.Fatalf("receive release: %v", err)
	}
	if err := session.Transition(netconn.EventDiffieRequested); err != nil {
		t.Fatalf("transition securing: %v", err)
	}
	if err := session.Receive(context.Background(), diffieCompletePacket(t)); err != nil {
		t.Fatalf("receive diffie complete: %v", err)
	}
	if session.State() != netconn.StateClosed {
		t.Fatalf("expected closed session, got %d", session.State())
	}
}

// testSession creates a handshake session.
func testSession(t *testing.T) *netconn.Session {
	t.Helper()
	return testSessionWithFactory(t, nil, nil)
}

// testSessionWithFactory creates a configurable handshake session.
func testSessionWithFactory(t *testing.T, factory *diffie.Factory, observe func(codec.Packet)) *netconn.Session {
	t.Helper()
	inbound := netconn.NewHandlerRegistry()
	outbound := netconn.NewHandlerRegistry()
	outbound.SetFallback(
		func(netconn.Context, codec.Packet) error {
			return nil
		},
		netconn.AllowAnyActiveState(),
		netconn.AllowUnauthenticated(),
	)
	Register(inbound, factory)
	session, err := netconn.NewSession(netconn.SessionConfig{
		ID:       "handshake-test",
		Kind:     "websocket",
		Inbound:  inbound,
		Outbound: outbound,
		Sender: func(_ context.Context, packet codec.Packet) error {
			if observe != nil {
				observe(packet)
			}
			return nil
		},
		Disposer: func(context.Context, netconn.Reason) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}

	return session
}

// testDiffieFactory generates an isolated RSA fixture for handler behavior.
func testDiffieFactory(t *testing.T) *diffie.Factory {
	t.Helper()
	factory, _ := testNegotiationFactory(t)
	return factory
}

// releasePacket creates a release-version packet.
func releasePacket(t *testing.T) codec.Packet {
	t.Helper()
	packet, err := codec.NewPacket(
		inrelease.Header,
		inrelease.Definition,
		codec.String("NITRO-test"),
		codec.String("HTML5"),
		codec.Int32(0),
		codec.Int32(0),
	)
	if err != nil {
		t.Fatalf("new release packet: %v", err)
	}

	return packet
}

// variablesPacket creates a variables packet.
func variablesPacket(t *testing.T) codec.Packet {
	t.Helper()
	packet, err := codec.NewPacket(invariables.Header, invariables.Definition, codec.Int32(1), codec.String("client"), codec.String("vars"))
	if err != nil {
		t.Fatalf("new variables packet: %v", err)
	}

	return packet
}

// policyPacket creates a policy packet.
func policyPacket(t *testing.T) codec.Packet {
	t.Helper()
	packet, err := codec.NewPacket(inpolicy.Header, inpolicy.Definition)
	if err != nil {
		t.Fatalf("new policy packet: %v", err)
	}

	return packet
}

// diffieInitPacket creates a Diffie init packet.
func diffieInitPacket(t *testing.T) codec.Packet {
	t.Helper()
	packet, err := codec.NewPacket(indiffieinit.Header, indiffieinit.Definition)
	if err != nil {
		t.Fatalf("new diffie init packet: %v", err)
	}

	return packet
}

// diffieCompletePacket creates a Diffie complete packet.
func diffieCompletePacket(t *testing.T) codec.Packet {
	t.Helper()
	packet, err := codec.NewPacket(indiffiecomplete.Header, indiffiecomplete.Definition, codec.String("key"))
	if err != nil {
		t.Fatalf("new diffie complete packet: %v", err)
	}

	return packet
}
