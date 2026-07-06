// Package live contains active room runtime state.
package live

import "errors"

var (
	// ErrInvalidRoom reports malformed active room data.
	ErrInvalidRoom = errors.New("invalid active room")

	// ErrRoomClosed reports a closed active room.
	ErrRoomClosed = errors.New("room closed")

	// ErrRoomNotFound reports a missing active room.
	ErrRoomNotFound = errors.New("active room not found")

	// ErrInvalidOccupant reports malformed occupant data.
	ErrInvalidOccupant = errors.New("invalid room occupant")

	// ErrRoomFull reports an active room at capacity.
	ErrRoomFull = errors.New("room full")
)
