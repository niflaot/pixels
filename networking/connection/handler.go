package connection

import (
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

// Handler emits commands for a packet handling event.
type Handler func(Context, codec.Packet) ([]Command, error)

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

// Handle emits commands for a packet using the matching handler.
func (registry *HandlerRegistry) Handle(context Context, packet codec.Packet) ([]Command, error) {
	registry.mutex.RLock()
	handler := registry.handlers[packet.Header]
	if handler == nil {
		handler = registry.fallback
	}
	registry.mutex.RUnlock()

	if handler == nil {
		return nil, ErrHandlerNotFound
	}

	return handler(context, packet)
}

// Len returns the number of registered header handlers.
func (registry *HandlerRegistry) Len() int {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	return len(registry.handlers)
}
