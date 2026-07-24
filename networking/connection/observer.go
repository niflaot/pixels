package connection

import (
	"sync"

	"github.com/niflaot/pixels/networking/codec"
)

// PacketObserver receives successful protocol traffic without changing routing.
type PacketObserver interface {
	// Observe receives one immutable connection snapshot and packet.
	Observe(Context, codec.Packet)
}

// PacketObserverFunc adapts a function to PacketObserver.
type PacketObserverFunc func(Context, codec.Packet)

// Observe calls the wrapped packet observer function.
func (observer PacketObserverFunc) Observe(context Context, packet codec.Packet) {
	observer(context, packet)
}

// ObserverRegistry broadcasts packet traffic to registered observers.
type ObserverRegistry struct {
	// mutex protects observer registration and snapshots.
	mutex sync.RWMutex
	// observers stores process-lifetime traffic observers.
	observers []PacketObserver
}

// NewObserverRegistry creates an empty packet observer registry.
func NewObserverRegistry() *ObserverRegistry {
	return &ObserverRegistry{}
}

// Register appends one packet observer.
func (registry *ObserverRegistry) Register(observer PacketObserver) error {
	if registry == nil || observer == nil {
		return ErrInvalidHandler
	}

	registry.mutex.Lock()
	registry.observers = append(registry.observers, observer)
	registry.mutex.Unlock()

	return nil
}

// Notify sends one packet to every registered observer.
func (registry *ObserverRegistry) Notify(context Context, packet codec.Packet) {
	if registry == nil {
		return
	}

	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	for _, observer := range registry.observers {
		observer.Observe(context, packet)
	}
}

// Len returns the number of registered observers.
func (registry *ObserverRegistry) Len() int {
	if registry == nil {
		return 0
	}

	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	return len(registry.observers)
}
