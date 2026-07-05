package connection

import (
	"context"
	"errors"
	"testing"
)

// TestDisconnectCodeString verifies stable disconnection code labels.
func TestDisconnectCodeString(t *testing.T) {
	cases := map[DisconnectCode]string{
		DisconnectUnknown:               "unknown",
		DisconnectLocalClose:            "local_close",
		DisconnectRemoteClose:           "remote_close",
		DisconnectTransportError:        "transport_error",
		DisconnectProtocolError:         "protocol_error",
		DisconnectAuthenticationFailed:  "authentication_failed",
		DisconnectAuthenticationTimeout: "authentication_timeout",
		DisconnectDuplicateSession:      "duplicate_session",
		DisconnectIdleTimeout:           "idle_timeout",
		DisconnectRateLimited:           "rate_limited",
		DisconnectPolicyViolation:       "policy_violation",
		DisconnectKicked:                "kicked",
		DisconnectBanned:                "banned",
		DisconnectServerShutdown:        "server_shutdown",
		DisconnectCode(99):              "unknown",
	}

	for code, expected := range cases {
		if code.String() != expected {
			t.Fatalf("expected %s, got %s", expected, code.String())
		}
	}
}

// TestUnknownReason verifies default unknown reason creation.
func TestUnknownReason(t *testing.T) {
	reason := UnknownReason()
	if reason.Code != DisconnectUnknown {
		t.Fatalf("expected unknown code, got %d", reason.Code)
	}
}

// TestSecurityPolicyForEnvironment verifies environment policy defaults.
func TestSecurityPolicyForEnvironment(t *testing.T) {
	if SecurityPolicyForEnvironment("development").Mode != SecurityOptional {
		t.Fatal("expected optional development security")
	}

	if SecurityPolicyForEnvironment("production").Mode != SecurityRequired {
		t.Fatal("expected required production security")
	}
}

// TestSessionTransitions verifies lifecycle transition validation.
func TestSessionTransitions(t *testing.T) {
	session := mustSession(t, sessionFixture(t))
	mustTransition(t, session, EventPacketReceived)
	mustTransition(t, session, EventDiffieRequested)
	mustTransition(t, session, EventDiffieCompleted)
	mustTransition(t, session, EventAuthenticationStarted)
	mustTransition(t, session, EventAuthenticationAccepted)
	mustTransition(t, session, EventSessionReady)

	if session.State() != StateConnected {
		t.Fatalf("expected connected state, got %d", session.State())
	}

	err := session.Transition(EventAuthenticationStarted)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

// TestSessionSecurityOpenSeal verifies ready secure channel byte wrapping.
func TestSessionSecurityOpenSeal(t *testing.T) {
	session := mustSession(t, sessionFixture(t))
	channel := &fakeSecureChannel{state: SecurityReady}
	if err := session.AttachSecurity(channel); err != nil {
		t.Fatalf("attach security: %v", err)
	}

	opened, err := session.Open([]byte("wire"))
	if err != nil {
		t.Fatalf("open bytes: %v", err)
	}

	if string(opened) != "open:wire" {
		t.Fatalf("expected opened bytes, got %s", opened)
	}

	sealed, err := session.Seal([]byte("plain"))
	if err != nil {
		t.Fatalf("seal bytes: %v", err)
	}

	if string(sealed) != "seal:plain" {
		t.Fatalf("expected sealed bytes, got %s", sealed)
	}
}

// TestSessionProductionAuthenticationRequiresSecurity verifies required security.
func TestSessionProductionAuthenticationRequiresSecurity(t *testing.T) {
	fixture := sessionFixture(t)
	fixture.SecurityPolicy = SecurityPolicy{Mode: SecurityRequired}
	session := mustSession(t, fixture)
	err := session.ValidateAuthenticationSecurity(context.Background())
	if !errors.Is(err, ErrSecurityRequired) {
		t.Fatalf("expected security required, got %v", err)
	}

	if session.State() != StateClosed {
		t.Fatalf("expected closed state, got %d", session.State())
	}
}

// TestSessionProductionAuthenticationAllowsReadySecurity verifies ready security.
func TestSessionProductionAuthenticationAllowsReadySecurity(t *testing.T) {
	fixture := sessionFixture(t)
	fixture.SecurityPolicy = SecurityPolicy{Mode: SecurityRequired}
	session := mustSession(t, fixture)
	if err := session.AttachSecurity(&fakeSecureChannel{state: SecurityReady}); err != nil {
		t.Fatalf("attach security: %v", err)
	}

	if err := session.ValidateAuthenticationSecurity(context.Background()); err != nil {
		t.Fatalf("validate authentication security: %v", err)
	}
}

// fakeSecureChannel records byte wrapping calls.
type fakeSecureChannel struct {
	state SecurityState
}

// State returns the fake security phase.
func (channel *fakeSecureChannel) State() SecurityState {
	return channel.state
}

// Begin starts fake security negotiation.
func (channel *fakeSecureChannel) Begin(context.Context) error {
	channel.state = SecurityNegotiating

	return nil
}

// Open unwraps fake inbound bytes.
func (channel *fakeSecureChannel) Open(src []byte) ([]byte, error) {
	return append([]byte("open:"), src...), nil
}

// Seal wraps fake outbound bytes.
func (channel *fakeSecureChannel) Seal(src []byte) ([]byte, error) {
	return append([]byte("seal:"), src...), nil
}

// Close releases fake security state.
func (channel *fakeSecureChannel) Close(Reason) error {
	channel.state = SecurityFailed

	return nil
}
