package connection

import (
	"context"
	"sync"
	"time"

	"github.com/niflaot/pixels/networking/codec"
)

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
	state            State
	trafficStarted   bool
	policy           SecurityPolicy
	security         SecureChannel
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
		state:     StateCreated,
		policy:    normalizeSecurityPolicy(config.SecurityPolicy),
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

	if !canTransition(session.state, EventAuthenticationAccepted) {
		return ErrInvalidTransition
	}

	if at.IsZero() {
		at = time.Now()
	}

	session.authenticatedAt = at
	session.authenticated = true
	session.state = StateAuthenticated

	return nil
}

// Receive handles an inbound packet.
func (session *Session) Receive(ctx context.Context, packet codec.Packet) error {
	if err := session.markTraffic(EventPacketReceived); err != nil {
		return err
	}

	context := session.context(InboundDirection)
	if context.Disconnected {
		return ErrDisposed
	}

	return session.inbound.Handle(context, packet)
}

// Send handles and writes an outbound packet.
func (session *Session) Send(ctx context.Context, packet codec.Packet) error {
	if err := session.markTraffic(""); err != nil {
		return err
	}

	context := session.context(OutboundDirection)
	if context.Disconnected {
		return ErrDisposed
	}

	if err := session.outbound.Handle(context, packet); err != nil {
		return err
	}

	return session.sender(ctx, packet)
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
	session.state = StateClosing
	close(session.done)
	disposer := session.disposer
	security := session.security
	session.mutex.Unlock()

	if security != nil {
		_ = security.Close(reason)
	}

	err := disposer(ctx, reason)

	session.mutex.Lock()
	session.state = StateClosed
	session.mutex.Unlock()

	return err
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
		State:            session.state,
		StartedAt:        session.startedAt,
		AuthenticatedAt:  session.authenticatedAt,
		Authenticated:    session.authenticated,
		Disconnected:     session.disconnected,
		DisconnectReason: session.disconnectReason,
	}
}
