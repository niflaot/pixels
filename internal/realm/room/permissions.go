package room

import "github.com/niflaot/pixels/internal/permission"

var (
	// EnterAny allows a player to enter regardless of room access mode or ban.
	EnterAny = permission.RegisterNode("room.enter.any", "")

	// EnterFull allows a player to enter a room at its normal capacity.
	EnterFull = permission.RegisterNode("room.enter.full", "")

	// AnswerAnyDoorbell allows a player to answer doorbells in any occupied room.
	AnswerAnyDoorbell = permission.RegisterNode("room.doorbell.answer.any", "")
)
