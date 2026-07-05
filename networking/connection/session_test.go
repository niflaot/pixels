package connection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/niflaot/pixels/networking/codec"
)

// TestSessionReceiveEmitsCommands verifies inbound handling.
func TestSessionReceiveEmitsCommands(t *testing.T) {
	session := mustSession(t, sessionFixture(t))
	commands, err := session.Receive(context.Background(), codec.Packet{Header: 1})
	if err != nil {
		t.Fatalf("receive packet: %v", err)
	}

	if commands[0].Direction != InboundDirection {
		t.Fatalf("expected inbound direction, got %d", commands[0].Direction)
	}
}

// TestSessionSendWritesAndEmitsCommands verifies outbound handling and sending.
func TestSessionSendWritesAndEmitsCommands(t *testing.T) {
	fixture := sessionFixture(t)
	session := mustSession(t, fixture)
	commands, err := session.Send(context.Background(), codec.Packet{Header: 2})
	if err != nil {
		t.Fatalf("send packet: %v", err)
	}

	if commands[0].Direction != OutboundDirection {
		t.Fatalf("expected outbound direction, got %d", commands[0].Direction)
	}

	if *fixture.sent != 1 {
		t.Fatalf("expected %d sent packets, got %d", 1, *fixture.sent)
	}
}

// TestSessionAuthenticateTracksTime verifies authentication state.
func TestSessionAuthenticateTracksTime(t *testing.T) {
	session := mustSession(t, sessionFixture(t))
	authenticatedAt := time.Unix(20, 0)
	if err := session.Authenticate(authenticatedAt); err != nil {
		t.Fatalf("authenticate session: %v", err)
	}

	got, ok := session.AuthenticatedAt()
	if !ok {
		t.Fatal("expected authenticated session")
	}

	if !got.Equal(authenticatedAt) {
		t.Fatalf("expected %s, got %s", authenticatedAt, got)
	}
}

// TestSessionDisconnectDisposesOnce verifies disposal behavior.
func TestSessionDisconnectDisposesOnce(t *testing.T) {
	fixture := sessionFixture(t)
	session := mustSession(t, fixture)
	reason := Reason{Code: DisconnectLocalClose}

	if err := session.Disconnect(context.Background(), reason); err != nil {
		t.Fatalf("disconnect session: %v", err)
	}

	select {
	case <-session.Done():
	default:
		t.Fatal("expected done channel closed")
	}

	if *fixture.disposed != 1 {
		t.Fatalf("expected %d disposals, got %d", 1, *fixture.disposed)
	}

	if err := session.Disconnect(context.Background(), reason); !errors.Is(err, ErrDisposed) {
		t.Fatalf("expected disposed error, got %v", err)
	}
}

// TestSessionRejectsInvalidConfig verifies required transport callbacks.
func TestSessionRejectsInvalidConfig(t *testing.T) {
	_, err := NewSession(SessionConfig{})
	if !errors.Is(err, ErrInvalidConnectionConfig) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}

// TestSessionRejectsAfterDisconnect verifies disposed operation protection.
func TestSessionRejectsAfterDisconnect(t *testing.T) {
	session := mustSession(t, sessionFixture(t))
	if err := session.Disconnect(context.Background(), UnknownReason()); err != nil {
		t.Fatalf("disconnect session: %v", err)
	}

	_, err := session.Receive(context.Background(), codec.Packet{Header: 1})
	if !errors.Is(err, ErrDisposed) {
		t.Fatalf("expected disposed receive, got %v", err)
	}

	_, err = session.Send(context.Background(), codec.Packet{Header: 2})
	if !errors.Is(err, ErrDisposed) {
		t.Fatalf("expected disposed send, got %v", err)
	}

	if err := session.Authenticate(time.Now()); !errors.Is(err, ErrDisposed) {
		t.Fatalf("expected disposed authenticate, got %v", err)
	}
}

// sessionFixtureConfig extends session config with counters.
type sessionFixtureConfig struct {
	SessionConfig
	sent     *int
	disposed *int
}

// sessionFixture returns a configured test session fixture.
func sessionFixture(t *testing.T) sessionFixtureConfig {
	t.Helper()
	inbound := NewHandlerRegistry()
	outbound := NewHandlerRegistry()
	sent := 0
	disposed := 0

	mustRegister(t, inbound, 1, "inbound")
	mustRegister(t, outbound, 2, "outbound")

	return sessionFixtureConfig{
		SessionConfig: SessionConfig{
			ID:        "one",
			Kind:      "websocket",
			StartedAt: time.Unix(10, 0),
			Inbound:   inbound,
			Outbound:  outbound,
			Sender: func(context.Context, codec.Packet) error {
				sent++
				return nil
			},
			Disposer: func(context.Context, Reason) error {
				disposed++
				return nil
			},
		},
		sent:     &sent,
		disposed: &disposed,
	}
}

// mustRegister registers a packet handler or fails the test.
func mustRegister(t *testing.T, registry *HandlerRegistry, header uint16, name string) {
	t.Helper()
	handler := func(context Context, packet codec.Packet) ([]Command, error) {
		return []Command{NewCommand(name, context, packet, nil)}, nil
	}

	if err := registry.Register(header, handler); err != nil {
		t.Fatalf("register handler: %v", err)
	}
}

// mustSession creates a session or fails the test.
func mustSession(t *testing.T, config sessionFixtureConfig) *Session {
	t.Helper()
	session, err := NewSession(config.SessionConfig)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}

	return session
}
