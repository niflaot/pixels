package connection

// DisconnectCode names a protocol-agnostic disconnection reason.
type DisconnectCode uint16

const (
	// DisconnectUnknown is used when no better reason is known.
	DisconnectUnknown DisconnectCode = iota

	// DisconnectLocalClose is used when the server closes intentionally.
	DisconnectLocalClose

	// DisconnectRemoteClose is used when the peer closes intentionally.
	DisconnectRemoteClose

	// DisconnectTransportError is used when the transport fails.
	DisconnectTransportError

	// DisconnectProtocolError is used when packet framing or payloads are invalid.
	DisconnectProtocolError

	// DisconnectAuthenticationFailed is used when authentication is rejected.
	DisconnectAuthenticationFailed

	// DisconnectAuthenticationTimeout is used when authentication takes too long.
	DisconnectAuthenticationTimeout

	// DisconnectDuplicateSession is used when a newer session replaces this one.
	DisconnectDuplicateSession

	// DisconnectIdleTimeout is used when the connection is idle too long.
	DisconnectIdleTimeout

	// DisconnectRateLimited is used when the peer exceeds network limits.
	DisconnectRateLimited

	// DisconnectPolicyViolation is used when the peer violates server policy.
	DisconnectPolicyViolation

	// DisconnectKicked is used when moderation removes the peer.
	DisconnectKicked

	// DisconnectBanned is used when access is blocked by moderation.
	DisconnectBanned

	// DisconnectServerShutdown is used during controlled server shutdown.
	DisconnectServerShutdown
)

// Reason explains why a connection was disconnected.
type Reason struct {
	// Code is the stable disconnection category.
	Code DisconnectCode
	// Message adds optional transport or operator context.
	Message string
}

// String returns a stable reason label.
func (code DisconnectCode) String() string {
	switch code {
	case DisconnectLocalClose:
		return "local_close"
	case DisconnectRemoteClose:
		return "remote_close"
	case DisconnectTransportError:
		return "transport_error"
	case DisconnectProtocolError:
		return "protocol_error"
	case DisconnectAuthenticationFailed:
		return "authentication_failed"
	case DisconnectAuthenticationTimeout:
		return "authentication_timeout"
	case DisconnectDuplicateSession:
		return "duplicate_session"
	case DisconnectIdleTimeout:
		return "idle_timeout"
	case DisconnectRateLimited:
		return "rate_limited"
	case DisconnectPolicyViolation:
		return "policy_violation"
	case DisconnectKicked:
		return "kicked"
	case DisconnectBanned:
		return "banned"
	case DisconnectServerShutdown:
		return "server_shutdown"
	default:
		return "unknown"
	}
}

// UnknownReason returns a reason with the unknown disconnection code.
func UnknownReason() Reason {
	return Reason{Code: DisconnectUnknown}
}

// normalizeSecurityPolicy fills missing policy values.
func normalizeSecurityPolicy(policy SecurityPolicy) SecurityPolicy {
	if policy.Mode == 0 {
		return DefaultSecurityPolicy()
	}

	return policy
}
