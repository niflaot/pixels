package connection

import "testing"

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
