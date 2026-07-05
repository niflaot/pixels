// Package connection defines transport-agnostic connection sessions.
package connection

import (
	"context"
	"sync"
	"time"

	"github.com/niflaot/pixels/networking/codec"
)

// ID identifies one connection within a connection kind.
type ID string

// Kind classifies the transport or session family of a connection.
type Kind string

// Sender writes an outbound packet through a transport.
type Sender func(context.Context, codec.Packet) error

// Disposer releases transport resources for a connection.
type Disposer func(context.Context, Reason) error

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
	// Receive handles an inbound packet and returns emitted commands.
	Receive(context.Context, codec.Packet) ([]Command, error)
	// Send handles and writes an outbound packet.
	Send(context.Context, codec.Packet) ([]Command, error)
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
	// Sender writes outbound packets through the transport.
	Sender Sender
	// Disposer releases transport resources.
	Disposer Disposer
}

// Session is a transport-agnostic connection implementation.
type Session struct {
	mutex            sync.RWMutex
	id               ID
	kind             Kind
	startedAt        time.Time
	authenticatedAt  time.Time
	authenticated    bool
	disconnected     bool
	disconnectReason Reason
	done             chan struct{}
	inbound          *HandlerRegistry
	outbound         *HandlerRegistry
	sender           Sender
	disposer         Disposer
}

// NewSession creates a session connection.
func NewSession(config SessionConfig) (*Session, error) {
	if config.ID == "" || config.Kind == "" || config.Sender == nil || config.Disposer == nil {
		return nil, ErrInvalidConnectionConfig
	}

	startedAt := config.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	inbound := config.Inbound
	if inbound == nil {
		inbound = NewHandlerRegistry()
	}

	outbound := config.Outbound
	if outbound == nil {
		outbound = NewHandlerRegistry()
	}

	return &Session{
		id:        config.ID,
		kind:      config.Kind,
		startedAt: startedAt,
		done:      make(chan struct{}),
		inbound:   inbound,
		outbound:  outbound,
		sender:    config.Sender,
		disposer:  config.Disposer,
	}, nil
}

// ID returns the connection identifier.
func (session *Session) ID() ID {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	return session.id
}

// Kind returns the connection kind.
func (session *Session) Kind() Kind {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	return session.kind
}

// StartedAt returns the connection start time.
func (session *Session) StartedAt() time.Time {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	return session.startedAt
}

// AuthenticatedAt returns the authentication time when available.
func (session *Session) AuthenticatedAt() (time.Time, bool) {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	return session.authenticatedAt, session.authenticated
}

// Authenticate marks the connection as authenticated.
func (session *Session) Authenticate(at time.Time) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.disconnected {
		return ErrDisposed
	}

	if at.IsZero() {
		at = time.Now()
	}

	session.authenticatedAt = at
	session.authenticated = true

	return nil
}

// Receive handles an inbound packet and returns emitted commands.
func (session *Session) Receive(ctx context.Context, packet codec.Packet) ([]Command, error) {
	context := session.context(InboundDirection)
	if context.Disconnected {
		return nil, ErrDisposed
	}

	return session.inbound.Handle(context, packet)
}

// Send handles and writes an outbound packet.
func (session *Session) Send(ctx context.Context, packet codec.Packet) ([]Command, error) {
	context := session.context(OutboundDirection)
	if context.Disconnected {
		return nil, ErrDisposed
	}

	commands, err := session.outbound.Handle(context, packet)
	if err != nil {
		return nil, err
	}

	if err := session.sender(ctx, packet); err != nil {
		return nil, err
	}

	return commands, nil
}

// Disconnect disposes the connection with a reason.
func (session *Session) Disconnect(ctx context.Context, reason Reason) error {
	session.mutex.Lock()
	if session.disconnected {
		session.mutex.Unlock()
		return ErrDisposed
	}

	session.disconnected = true
	session.disconnectReason = reason
	close(session.done)
	disposer := session.disposer
	session.mutex.Unlock()

	return disposer(ctx, reason)
}

// Done returns a channel closed when the connection is disposed.
func (session *Session) Done() <-chan struct{} {
	return session.done
}

// context returns an immutable handler context snapshot.
func (session *Session) context(direction Direction) Context {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	return Context{
		ConnectionID:     session.id,
		ConnectionKind:   session.kind,
		Direction:        direction,
		StartedAt:        session.startedAt,
		AuthenticatedAt:  session.authenticatedAt,
		Authenticated:    session.authenticated,
		Disconnected:     session.disconnected,
		DisconnectReason: session.disconnectReason,
	}
}
