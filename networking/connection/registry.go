package connection

import (
	"context"
	"sync"
)

// Registry stores active connections grouped by kind.
type Registry struct {
	mutex       sync.RWMutex
	connections map[Kind]map[ID]Connection
}

// NewRegistry creates an empty connection registry.
func NewRegistry() *Registry {
	return &Registry{connections: make(map[Kind]map[ID]Connection)}
}

// Register adds a connection to its kind bucket.
func (registry *Registry) Register(connection Connection) error {
	if connection == nil || connection.ID() == "" || connection.Kind() == "" {
		return ErrInvalidConnection
	}

	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	kind := connection.Kind()
	id := connection.ID()
	bucket := registry.connections[kind]
	if bucket == nil {
		bucket = make(map[ID]Connection)
		registry.connections[kind] = bucket
	}

	if _, exists := bucket[id]; exists {
		return ErrConnectionExists
	}

	bucket[id] = connection

	return nil
}

// Remove deletes a connection from its kind bucket.
func (registry *Registry) Remove(kind Kind, id ID) (Connection, bool) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	bucket := registry.connections[kind]
	if bucket == nil {
		return nil, false
	}

	connection, exists := bucket[id]
	if !exists {
		return nil, false
	}

	delete(bucket, id)
	if len(bucket) == 0 {
		delete(registry.connections, kind)
	}

	return connection, true
}

// Get returns a connection by kind and id.
func (registry *Registry) Get(kind Kind, id ID) (Connection, bool) {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	connection, exists := registry.connections[kind][id]

	return connection, exists
}

// List returns connections registered for one kind.
func (registry *Registry) List(kind Kind) []Connection {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	bucket := registry.connections[kind]
	connections := make([]Connection, 0, len(bucket))
	for _, connection := range bucket {
		connections = append(connections, connection)
	}

	return connections
}

// Count returns the number of connections in one kind bucket.
func (registry *Registry) Count(kind Kind) int {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	return len(registry.connections[kind])
}

// CountAll returns the number of registered connections.
func (registry *Registry) CountAll() int {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	count := 0
	for _, bucket := range registry.connections {
		count += len(bucket)
	}

	return count
}

// Disconnect removes and disconnects one connection.
func (registry *Registry) Disconnect(ctx context.Context, kind Kind, id ID, reason Reason) error {
	connection, exists := registry.Remove(kind, id)
	if !exists {
		return ErrConnectionNotFound
	}

	return connection.Disconnect(ctx, reason)
}

// DisconnectKind removes and disconnects every connection of a kind.
func (registry *Registry) DisconnectKind(ctx context.Context, kind Kind, reason Reason) []error {
	connections := registry.List(kind)
	errors := make([]error, 0)
	for _, connection := range connections {
		registry.Remove(kind, connection.ID())
		if err := connection.Disconnect(ctx, reason); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// DisconnectAll removes and disconnects every registered connection.
func (registry *Registry) DisconnectAll(ctx context.Context, reason Reason) []error {
	registry.mutex.RLock()
	connections := make([]Connection, 0)
	for _, bucket := range registry.connections {
		for _, connection := range bucket {
			connections = append(connections, connection)
		}
	}
	registry.mutex.RUnlock()

	errors := make([]error, 0)
	for _, connection := range connections {
		registry.Remove(connection.Kind(), connection.ID())
		if err := connection.Disconnect(ctx, reason); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
