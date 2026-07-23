package connection

import (
	"context"
	"testing"
	"time"
)

// TestSessionPolicyAndHeartbeatMethods verifies session policy and heartbeat helpers.
func TestSessionPolicyAndHeartbeatMethods(t *testing.T) {
	session := mustSession(t, sessionFixture(t))
	if err := session.SetSecurityPolicy(SecurityPolicy{Mode: SecurityRequired}); err != nil {
		t.Fatalf("set security policy: %v", err)
	}

	if session.SecurityPolicy().Mode != SecurityRequired {
		t.Fatal("expected required security policy")
	}

	now := time.Unix(50, 0)
	if err := session.MarkPong(now); err != nil {
		t.Fatalf("mark pong: %v", err)
	}

	if !session.LastPongAt().Equal(now) {
		t.Fatalf("expected pong time %s, got %s", now, session.LastPongAt())
	}

	activity := session.LastActivityAt()
	if err := session.Receive(context.Background(), codecPacket(1)); err != nil {
		t.Fatalf("receive packet: %v", err)
	}
	if !session.LastActivityAt().After(activity) {
		t.Fatalf("expected inbound activity after %s, got %s", activity, session.LastActivityAt())
	}

	if err := session.SetSecurityPolicy(SecurityPolicy{Mode: SecurityOptional}); err != ErrInvalidState {
		t.Fatalf("expected invalid state, got %v", err)
	}
}

// TestSessionActivatesAttachedNegotiatingChannel verifies the compatibility barrier path.
func TestSessionActivatesAttachedNegotiatingChannel(t *testing.T) {
	session := mustSession(t, sessionFixture(t))
	channel := &fakeActivatableChannel{fakeSecureChannel: fakeSecureChannel{state: SecurityPlain}}
	if err := session.BeginSecurity(context.Background(), channel); err != nil {
		t.Fatalf("begin security: %v", err)
	}
	if err := session.CompleteSecurity(context.Background(), codecPacket(2), channel); err != nil {
		t.Fatalf("complete security: %v", err)
	}
	if channel.State() != SecurityReady {
		t.Fatalf("expected ready channel, got %d", channel.State())
	}
}
