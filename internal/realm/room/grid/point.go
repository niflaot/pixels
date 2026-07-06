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
