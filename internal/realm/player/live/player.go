package live

import "sync"

// Player is the live runtime player composition root.
type Player struct {
	// mutex protects runtime snapshot replacement.
	mutex sync.RWMutex

	// snapshot stores durable player data copied into runtime state.
	snapshot Snapshot

	// peer stores the authenticated connection binding.
	peer SessionPeer
}

// NewPlayer creates a live player.
func NewPlayer(snapshot Snapshot, peer SessionPeer) (*Player, error) {
	if !snapshot.Valid() {
		return nil, ErrInvalidPlayer
	}
	if !peer.Valid() {
		return nil, ErrInvalidPeer
	}

	return &Player{snapshot: snapshot, peer: peer}, nil
}

// ID returns the player id.
func (player *Player) ID() int64 {
	player.mutex.RLock()
	defer player.mutex.RUnlock()

	return player.snapshot.ID
}

// Username returns the player username.
func (player *Player) Username() string {
	player.mutex.RLock()
	defer player.mutex.RUnlock()

	return player.snapshot.Username
}

// Snapshot returns a copy of the runtime player snapshot.
func (player *Player) Snapshot() Snapshot {
	player.mutex.RLock()
	defer player.mutex.RUnlock()

	return player.snapshot
}

// ReplaceSnapshot replaces durable runtime data.
func (player *Player) ReplaceSnapshot(snapshot Snapshot) error {
	if !snapshot.Valid() || snapshot.ID != player.ID() {
		return ErrInvalidPlayer
	}

	player.mutex.Lock()
	defer player.mutex.Unlock()

	player.snapshot = snapshot

	return nil
}

// Peer returns the player session peer.
func (player *Player) Peer() SessionPeer {
	player.mutex.RLock()
	defer player.mutex.RUnlock()

	return player.peer
}
