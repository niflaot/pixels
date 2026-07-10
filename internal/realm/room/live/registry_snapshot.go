package live

// Count returns the number of active rooms.
func (registry *Registry) Count() int {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	return len(registry.rooms)
}

// Snapshot returns stable active room references.
func (registry *Registry) Snapshot() []*Room {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	rooms := make([]*Room, 0, len(registry.rooms))
	for _, room := range registry.rooms {
		rooms = append(rooms, room)
	}

	return rooms
}
