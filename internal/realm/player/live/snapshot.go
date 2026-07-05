package live

import (
	playermodel "github.com/niflaot/pixels/internal/realm/player/model"
	playerservice "github.com/niflaot/pixels/internal/realm/player/service"
)

// Snapshot stores durable player data needed by runtime state.
type Snapshot struct {
	// ID identifies the player.
	ID int64

	// Username stores the visible player name.
	Username string

	// Look stores the avatar figure string.
	Look string

	// Gender stores the avatar gender code.
	Gender playermodel.Gender

	// Motto stores the public profile motto.
	Motto string

	// HomeRoomID stores the optional home room id.
	HomeRoomID *int64

	// AllowNameChange reports whether username changes are allowed.
	AllowNameChange bool
}

// SnapshotFromRecord maps a persistent player record to a runtime snapshot.
func SnapshotFromRecord(record playerservice.Record) Snapshot {
	return Snapshot{
		ID:              record.Player.ID,
		Username:        record.Player.Username,
		Look:            record.Profile.Look,
		Gender:          record.Profile.Gender,
		Motto:           record.Profile.Motto,
		HomeRoomID:      record.Profile.HomeRoomID,
		AllowNameChange: record.Profile.AllowNameChange,
	}
}

// Valid reports whether the snapshot can create a live player.
func (snapshot Snapshot) Valid() bool {
	return snapshot.ID > 0 && snapshot.Username != ""
}
