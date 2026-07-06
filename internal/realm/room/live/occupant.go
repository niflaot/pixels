package live

import (
	"time"

	netconn "github.com/niflaot/pixels/networking/connection"
)

// Occupant describes one player inside an active room.
type Occupant struct {
	// PlayerID identifies the player.
	PlayerID int64

	// Username stores a display snapshot for diagnostics.
	Username string

	// ConnectionID identifies the active connection.
	ConnectionID netconn.ID

	// ConnectionKind identifies the active connection family.
	ConnectionKind netconn.Kind

	// JoinedAt stores when the player joined the active room.
	JoinedAt time.Time
}

// Valid reports whether the occupant can join a room.
func (occupant Occupant) Valid() bool {
	return occupant.PlayerID > 0 && occupant.ConnectionID != "" && occupant.ConnectionKind != ""
}

// WithJoinTime returns the occupant with a default join time.
func (occupant Occupant) WithJoinTime(now time.Time) Occupant {
	if occupant.JoinedAt.IsZero() {
		occupant.JoinedAt = now
	}

	return occupant
}
