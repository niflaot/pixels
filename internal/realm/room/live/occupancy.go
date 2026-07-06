package live

// Occupancy describes active room population.
type Occupancy struct {
	// RoomID identifies the room.
	RoomID int64

	// CategoryID optionally identifies the room category.
	CategoryID *int64

	// Count stores the active occupancy count.
	Count int

	// MaxUsers stores the maximum active occupancy.
	MaxUsers int

	// PlayerIDs stores active player ids.
	PlayerIDs []int64
}
