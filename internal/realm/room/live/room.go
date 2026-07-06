package live

import (
	"sync"
	"time"
)

// Room stores active runtime state for one loaded room.
type Room struct {
	// mutex protects active room state.
	mutex sync.RWMutex

	// snapshot stores stable room metadata.
	snapshot Snapshot

	// occupants stores occupants by player id.
	occupants map[int64]Occupant

	// loadedAt stores when the active room was loaded.
	loadedAt time.Time

	// idleSince stores when the room became empty.
	idleSince *time.Time

	// closed reports whether the active room was closed.
	closed bool
}

// NewRoom creates an active room.
func NewRoom(snapshot Snapshot) (*Room, error) {
	if !snapshot.Valid() {
		return nil, ErrInvalidRoom
	}

	return &Room{snapshot: snapshot, occupants: make(map[int64]Occupant), loadedAt: time.Now()}, nil
}

// ID returns the room id.
func (room *Room) ID() int64 {
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	return room.snapshot.ID
}

// Snapshot returns active room metadata.
func (room *Room) Snapshot() Snapshot {
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	return room.snapshot
}

// Join adds or replaces an active room occupant.
func (room *Room) Join(occupant Occupant) (Occupancy, error) {
	if !occupant.Valid() {
		return Occupancy{}, ErrInvalidOccupant
	}

	room.mutex.Lock()
	defer room.mutex.Unlock()

	if room.closed {
		return Occupancy{}, ErrRoomClosed
	}
	if _, exists := room.occupants[occupant.PlayerID]; !exists && len(room.occupants) >= room.snapshot.MaxUsers {
		return Occupancy{}, ErrRoomFull
	}

	room.occupants[occupant.PlayerID] = occupant.WithJoinTime(time.Now())
	room.idleSince = nil

	return room.occupancyLocked(), nil
}

// Leave removes an active room occupant.
func (room *Room) Leave(playerID int64) (Occupancy, bool) {
	room.mutex.Lock()
	defer room.mutex.Unlock()

	if _, found := room.occupants[playerID]; !found {
		return Occupancy{}, false
	}

	delete(room.occupants, playerID)
	if len(room.occupants) == 0 {
		now := time.Now()
		room.idleSince = &now
	}

	return room.occupancyLocked(), true
}

// Occupancy returns a stable occupancy snapshot.
func (room *Room) Occupancy() Occupancy {
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	return room.occupancyLocked()
}

// Occupants returns a stable occupant snapshot.
func (room *Room) Occupants() []Occupant {
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	occupants := make([]Occupant, 0, len(room.occupants))
	for _, occupant := range room.occupants {
		occupants = append(occupants, occupant)
	}

	return occupants
}

// Close marks the active room as closed.
func (room *Room) Close() Occupancy {
	room.mutex.Lock()
	defer room.mutex.Unlock()

	room.closed = true
	room.occupants = make(map[int64]Occupant)
	now := time.Now()
	room.idleSince = &now

	return room.occupancyLocked()
}

// IdleSince returns when the room became empty.
func (room *Room) IdleSince() *time.Time {
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	if room.idleSince == nil {
		return nil
	}

	idleSince := *room.idleSince

	return &idleSince
}

// occupancyLocked returns occupancy while a room lock is held.
func (room *Room) occupancyLocked() Occupancy {
	playerIDs := make([]int64, 0, len(room.occupants))
	for playerID := range room.occupants {
		playerIDs = append(playerIDs, playerID)
	}

	return Occupancy{RoomID: room.snapshot.ID, CategoryID: room.snapshot.CategoryID, Count: len(room.occupants), MaxUsers: room.snapshot.MaxUsers, PlayerIDs: playerIDs}
}
