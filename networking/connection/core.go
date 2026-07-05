// Package connection defines transport-agnostic connection sessions.
package connection

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/niflaot/pixels/networking/codec"
)

var (
	// ErrConnectionExists reports an already registered connection id for a kind.
	ErrConnectionExists = errors.New("connection exists")

	// ErrConnectionNotFound reports a missing connection id for a kind.
	ErrConnectionNotFound = errors.New("connection not found")

	// ErrDisposed reports an operation attempted after disposal.
	ErrDisposed = errors.New("connection disposed")

	// ErrHandlerExists reports an already registered handler for a packet header.
	ErrHandlerExists = errors.New("handler exists")

	// ErrHandlerNotFound reports a missing handler for a packet header.
	ErrHandlerNotFound = errors.New("handler not found")

	// ErrInvalidConnection reports an invalid connection value.
	ErrInvalidConnection = errors.New("invalid connection")

	// ErrInvalidConnectionConfig reports an invalid session configuration.
	ErrInvalidConnectionConfig = errors.New("invalid connection config")

	// ErrInvalidHandler reports an invalid packet handler.
	ErrInvalidHandler = errors.New("invalid handler")

	// ErrInvalidSecurity reports an invalid secure channel.
	ErrInvalidSecurity = errors.New("invalid security")

	// ErrInvalidState reports an invalid state operation.
	ErrInvalidState = errors.New("invalid state")

	// ErrInvalidTransition reports an invalid state transition.
	ErrInvalidTransition = errors.New("invalid transition")

	// ErrSecurityRequired reports missing required security.
	ErrSecurityRequired = errors.New("security required")
)

// ID identifies one connection within a connection kind.
type ID string

// Kind classifies the transport or session family of a connection.
type Kind string

// Sender writes an outbound packet through a transport.
type Sender func(context.Context, codec.Packet) error

// Disposer releases transport resources for a connection.
type Disposer func(context.Context, Reason) error

// Direction names whether a packet is entering or leaving a connection.
type Direction uint8

const (
	// InboundDirection describes a packet received from the peer.
	InboundDirection Direction = iota + 1

	// OutboundDirection describes a packet sent to the peer.
	OutboundDirection
)

// SecurityState names the byte security phase.
type SecurityState uint8

const (
	// SecurityPlain means traffic is not encrypted.
	SecurityPlain SecurityState = iota + 1

	// SecurityNegotiating means security is being negotiated.
	SecurityNegotiating

	// SecurityReady means encryption can open and seal bytes.
	SecurityReady

	// SecurityFailed means security negotiation failed.
	SecurityFailed
)

// SecurityMode names whether encryption is required.
type SecurityMode uint8

const (
	// SecurityOptional allows plain traffic before authentication.
	SecurityOptional SecurityMode = iota + 1

	// SecurityRequired requires secure traffic before authentication.
	SecurityRequired
)

// SecurityPolicy controls connection security requirements.
type SecurityPolicy struct {
	// Mode names whether security is optional or required.
	Mode SecurityMode
}

// SecureChannel opens and seals transport bytes for a session.
type SecureChannel interface {
	// State returns the security phase.
	State() SecurityState
	// Begin starts security negotiation.
	Begin(context.Context) error
	// Open unwraps inbound bytes.
	Open([]byte) ([]byte, error)
	// Seal wraps outbound bytes.
	Seal([]byte) ([]byte, error)
	// Close releases security state.
	Close(Reason) error
}

// DefaultSecurityPolicy returns the development-friendly policy.
func DefaultSecurityPolicy() SecurityPolicy {
	return SecurityPolicy{Mode: SecurityOptional}
}

// SecurityPolicyForEnvironment returns a policy for an environment name.
func SecurityPolicyForEnvironment(environment string) SecurityPolicy {
	if strings.EqualFold(environment, "production") {
		return SecurityPolicy{Mode: SecurityRequired}
	}

	return DefaultSecurityPolicy()
}

// Connection describes one transport-agnostic session.
type Connection interface {
	// ID returns the connection identifier.
	ID() ID
	// Kind returns the connection kind.
	Kind() Kind
	// StartedAt returns the connection start time.
	StartedAt() time.Time
	// AuthenticatedAt returns the authentication time when available.
	AuthenticatedAt() (time.Time, bool)
	// Authenticate marks the connection as authenticated.
	Authenticate(time.Time) error
	// Receive handles an inbound packet.
	Receive(context.Context, codec.Packet) error
	// Send handles and writes an outbound packet.
	Send(context.Context, codec.Packet) error
	// Disconnect disposes the connection with a reason.
	Disconnect(context.Context, Reason) error
	// Done returns a channel closed when the connection is disposed.
	Done() <-chan struct{}
}

// SessionConfig configures a session connection.
type SessionConfig struct {
	// ID identifies one connection within its kind.
	ID ID
	// Kind classifies the connection transport family.
	Kind Kind
	// StartedAt overrides the connection start time.
	StartedAt time.Time
	// Inbound handles packets received from the peer.
	Inbound *HandlerRegistry
	// Outbound handles packets sent to the peer.
	Outbound *HandlerRegistry
	// SecurityPolicy controls whether encryption is required.
	SecurityPolicy SecurityPolicy
	// Sender writes outbound packets through the transport.
	Sender Sender
	// Disposer releases transport resources.
	Disposer Disposer
}
