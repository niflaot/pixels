// Package walk moves a live room unit toward a target tile.
package walk

import (
	"context"
	"errors"

	"github.com/niflaot/pixels/internal/command"
	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	roomsession "github.com/niflaot/pixels/internal/realm/room/commands/session"
	roomlive "github.com/niflaot/pixels/internal/realm/room/live"
	"github.com/niflaot/pixels/internal/realm/room/world/grid"
	"github.com/niflaot/pixels/internal/realm/session/binding"
	netconn "github.com/niflaot/pixels/networking/connection"
)

const (
	// Name identifies the room walk command.
	Name command.Name = "room.walk"
)

var (
	// ErrPlayerNotInRoom reports a walk without active room presence.
	ErrPlayerNotInRoom = errors.New("player not in room")

	// ErrInvalidTarget reports malformed target coordinates.
	ErrInvalidTarget = errors.New("invalid walk target")
)

// Command moves a player unit.
type Command struct {
	// Handler stores the source connection handler.
	Handler netconn.Context

	// X stores the target tile x coordinate.
	X int

	// Y stores the target tile y coordinate.
	Y int
}

// Handler handles room walk commands.
type Handler struct {
	// Players stores live player state.
	Players *playerlive.Registry

	// Bindings stores player connection bindings.
	Bindings *binding.Registry

	// Runtime stores active rooms.
	Runtime *roomlive.Registry
}

// CommandName returns the stable command name.
func (Command) CommandName() command.Name {
	return Name
}

// Handle handles a room walk command.
func (handler Handler) Handle(ctx context.Context, envelope command.Envelope[Command]) error {
	player, err := roomsession.Player(envelope.Command.Handler, handler.Bindings, handler.Players)
	if err != nil {
		return err
	}
	roomID, found := player.CurrentRoom()
	if !found {
		return ErrPlayerNotInRoom
	}
	active, found := handler.Runtime.Find(roomID)
	if !found {
		return roomlive.ErrRoomNotFound
	}
	point, ok := grid.NewPoint(envelope.Command.X, envelope.Command.Y)
	if !ok {
		return ErrInvalidTarget
	}

	_, err = active.MoveTo(player.ID(), point)

	return err
}
