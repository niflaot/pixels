package grid

// TileFlag stores compact tile metadata.
type TileFlag uint8

const (
	// FlagInvalid marks a tile that does not exist in the room.
	FlagInvalid TileFlag = 1 << iota

	// FlagDoor marks the room door tile.
	FlagDoor
)

// Tile stores a compact immutable view of a room tile.
type Tile struct {
	// point stores the tile coordinate.
	point Point

	// height stores the base tile height.
	height Height

	// flags stores compact tile metadata.
	flags TileFlag
}

// Point returns the tile coordinate.
func (tile Tile) Point() Point {
	return tile.point
}

// Height returns the tile base height.
func (tile Tile) Height() Height {
	return tile.height
}

// Flags returns the tile flags.
func (tile Tile) Flags() TileFlag {
	return tile.flags
}

// Valid reports whether the tile exists.
func (tile Tile) Valid() bool {
	return tile.flags&FlagInvalid == 0
}

// Door reports whether the tile is the room door.
func (tile Tile) Door() bool {
	return tile.flags&FlagDoor != 0
}
