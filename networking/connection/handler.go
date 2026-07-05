package connection

import (
	"context"
	"sync"
	"time"

	"github.com/niflaot/pixels/networking/codec"
)

// Context describes the connection state visible to packet handlers.
type Context struct {
	// ConnectionID is the handled connection identifier.
	ConnectionID ID
	// ConnectionKind is the handled connection kind.
	ConnectionKind Kind
	// Direction is the packet flow direction.
	Direction Direction
	// State is the connection lifecycle phase.
	State State
	// StartedAt is the connection start time.
	StartedAt time.Time
	// AuthenticatedAt is the authentication time when authenticated.
	AuthenticatedAt time.Time
	// Authenticated reports whether authentication completed.
	Authenticated bool
	// Disconnected reports whether the connection is disposed.
	Disconnected bool
	// DisconnectReason stores the disposal reason when disconnected.
	DisconnectReason Reason
}

// Handler executes realm-owned packet behavior.
type Handler func(Context, codec.Packet) error

// HandlerRegistry stores packet handlers by header.
type HandlerRegistry struct {
	mutex    sync.RWMutex
	handlers map[uint16]Handler
	fallback Handler
}

// NewHandlerRegistry creates an empty handler registry.
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: make(map[uint16]Handler)}
}

// Register adds a handler for a packet header.
func (registry *HandlerRegistry) Register(header uint16, handler Handler) error {
	if handler == nil {
		return ErrInvalidHandler
	}

	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	if _, exists := registry.handlers[header]; exists {
		return ErrHandlerExists
	}

	registry.handlers[header] = handler

	return nil
}

// Unregister removes a handler for a packet header.
func (registry *HandlerRegistry) Unregister(header uint16) bool {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	if _, exists := registry.handlers[header]; !exists {
		return false
	}

	delete(registry.handlers, header)

	return true
}

// SetFallback changes the handler used when no header handler is registered.
func (registry *HandlerRegistry) SetFallback(handler Handler) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	registry.fallback = handler
}

// Handle routes a packet to the matching handler.
func (registry *HandlerRegistry) Handle(context Context, packet codec.Packet) error {
	registry.mutex.RLock()
	handler := registry.handlers[packet.Header]
	if handler == nil {
		handler = registry.fallback
	}
	registry.mutex.RUnlock()

	if handler == nil {
		return ErrHandlerNotFound
	}

	return handler(context, packet)
}

// Len returns the number of registered header handlers.
func (registry *HandlerRegistry) Len() int {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	return len(registry.handlers)
}

// SecurityPolicy returns the connection security policy.
func (session *Session) SecurityPolicy() SecurityPolicy {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	return session.policy
}

// SetSecurityPolicy changes security policy before traffic starts.
func (session *Session) SetSecurityPolicy(policy SecurityPolicy) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.trafficStarted {
		return ErrInvalidState
	}

	session.policy = normalizeSecurityPolicy(policy)

	return nil
}

// AttachSecurity attaches a secure channel to the session.
func (session *Session) AttachSecurity(channel SecureChannel) error {
	if channel == nil {
		return ErrInvalidSecurity
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.disconnected {
		return ErrDisposed
	}

	if session.security != nil {
		return ErrInvalidSecurity
	}

	session.security = channel

	return nil
}

// SecurityState returns the attached secure channel state.
func (session *Session) SecurityState() SecurityState {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	if session.security == nil {
		return SecurityPlain
	}

	return session.security.State()
}

// Open unwraps inbound bytes when security is ready.
func (session *Session) Open(src []byte) ([]byte, error) {
	channel := session.secureChannel()
	if channel == nil || channel.State() != SecurityReady {
		return src, nil
	}

	return channel.Open(src)
}

// Seal wraps outbound bytes when security is ready.
func (session *Session) Seal(src []byte) ([]byte, error) {
	channel := session.secureChannel()
	if channel == nil || channel.State() != SecurityReady {
		return src, nil
	}

	return channel.Seal(src)
}

// ValidateAuthenticationSecurity checks security before authentication.
func (session *Session) ValidateAuthenticationSecurity(ctx context.Context) error {
	if session.SecurityPolicy().Mode != SecurityRequired {
		return nil
	}

	if session.SecurityState() == SecurityReady {
		return nil
	}

	_ = session.Transition(EventProtocolFailed)
	_ = session.Disconnect(ctx, Reason{Code: DisconnectProtocolError, Message: ErrSecurityRequired.Error()})

	return ErrSecurityRequired
}

// secureChannel returns the attached secure channel.
func (session *Session) secureChannel() SecureChannel {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	return session.security
}
