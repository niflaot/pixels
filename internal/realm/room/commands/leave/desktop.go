package leave

import (
	"context"
	"errors"

	"github.com/niflaot/pixels/internal/command"
	netconn "github.com/niflaot/pixels/networking/connection"
	outdesktop "github.com/niflaot/pixels/networking/outbound/session/desktop"
)

// ToDesktop leaves a room through the standard lifecycle and opens hotel view.
func (handler Handler) ToDesktop(ctx context.Context, playerID int64) error {
	target := handler.connection(playerID)
	leaveErr := handler.Handle(ctx, command.Envelope[Command]{Command: Command{PlayerID: playerID}})
	packet, err := outdesktop.Encode()
	if err != nil {
		return errors.Join(leaveErr, err)
	}
	if target == nil {
		return leaveErr
	}

	return errors.Join(leaveErr, target.Send(ctx, packet))
}

// connection resolves an occupant connection before room removal.
func (handler Handler) connection(playerID int64) netconn.Connection {
	if handler.Runtime == nil || handler.Connections == nil {
		return nil
	}
	active, found := handler.Runtime.FindByPlayer(playerID)
	if !found {
		return nil
	}
	for _, occupant := range active.Occupants() {
		if occupant.PlayerID == playerID {
			connection, _ := handler.Connections.Get(occupant.ConnectionKind, occupant.ConnectionID)

			return connection
		}
	}

	return nil
}
