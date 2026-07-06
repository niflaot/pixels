package live

// Snapshot stores room metadata needed by runtime occupancy.
type Snapshot struct {
	// ID identifies the room.
	ID int64

	// CategoryID optionally identifies the navigator category.
	CategoryID *int64

	// MaxUsers stores the maximum active occupancy.
	MaxUsers int
}

// Valid reports whether the snapshot can back an active room.
func (snapshot Snapshot) Valid() bool {
	return snapshot.ID > 0 && snapshot.MaxUsers > 0
}
