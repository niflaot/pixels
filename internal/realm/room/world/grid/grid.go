package grid

// Height stores a room height value in compact fixed-point units.
type Height int16

// Point stores tile coordinates in a room grid.
type Point struct {
	// X stores the horizontal coordinate.
	X uint16

	// Y stores the vertical coordinate.
	Y uint16
}

// NewPoint creates a point from signed coordinates.
func NewPoint(x int, y int) (Point, bool) {
	if x < 0 || y < 0 || x > int(^uint16(0)) || y > int(^uint16(0)) {
		return Point{}, false
	}

	return Point{X: uint16(x), Y: uint16(y)}, true
}

// MustPoint creates a point and panics when coordinates are invalid.
func MustPoint(x int, y int) Point {
	point, ok := NewPoint(x, y)
	if !ok {
		panic(ErrOutOfBounds)
	}

	return point
}

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

// Grid stores an immutable compact room heightmap.
type Grid struct {
	// width stores the number of columns.
	width uint16

	// height stores the number of rows.
	height uint16

	// heights stores base heights indexed by y*width+x.
	heights []Height

	// flags stores tile flags indexed by y*width+x.
	flags []TileFlag

	// door stores the room door coordinate.
	door Point

	// hasDoor reports whether a door was configured.
	hasDoor bool

	// validCount stores the number of existing tiles.
	validCount int
}

// Width returns the grid width.
func (grid Grid) Width() uint16 {
	return grid.width
}

// Height returns the grid height.
func (grid Grid) Height() uint16 {
	return grid.height
}

// TileCount returns the total number of grid tiles.
func (grid Grid) TileCount() int {
	return int(grid.width) * int(grid.height)
}

// ValidCount returns the number of existing tiles.
func (grid Grid) ValidCount() int {
	return grid.validCount
}

// Door returns the room door coordinate.
func (grid Grid) Door() (Point, bool) {
	return grid.door, grid.hasDoor
}

// InBounds reports whether a point belongs to the grid.
func (grid Grid) InBounds(point Point) bool {
	return point.X < grid.width && point.Y < grid.height
}

// Index returns the compact slice index for a point.
func (grid Grid) Index(point Point) (int, bool) {
	if !grid.InBounds(point) {
		return 0, false
	}

	return int(point.Y)*int(grid.width) + int(point.X), true
}

// Tile returns the tile at a point.
func (grid Grid) Tile(point Point) (Tile, bool) {
	index, ok := grid.Index(point)
	if !ok {
		return Tile{}, false
	}

	return Tile{point: point, height: grid.heights[index], flags: grid.flags[index]}, true
}

// HeightAt returns the base height at a point.
func (grid Grid) HeightAt(point Point) (Height, bool) {
	tile, ok := grid.Tile(point)
	if !ok || !tile.Valid() {
		return 0, false
	}

	return tile.Height(), true
}

// FlagsAt returns tile flags at a point.
func (grid Grid) FlagsAt(point Point) (TileFlag, bool) {
	index, ok := grid.Index(point)
	if !ok {
		return 0, false
	}

	return grid.flags[index], true
}

// Valid reports whether a point is inside the grid and exists.
func (grid Grid) Valid(point Point) bool {
	flags, ok := grid.FlagsAt(point)

	return ok && flags&FlagInvalid == 0
}
