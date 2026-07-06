package enter

import (
	"context"

	"github.com/niflaot/pixels/internal/realm/room/broadcast"
	roomlive "github.com/niflaot/pixels/internal/realm/room/live"
	"github.com/niflaot/pixels/internal/realm/room/projection"
	netconn "github.com/niflaot/pixels/networking/connection"
	outstatus "github.com/niflaot/pixels/networking/outbound/room/entities/status"
	outunits "github.com/niflaot/pixels/networking/outbound/room/entities/units"
)

// sendRoomState sends the current visible room units to one connection.
func (handler Handler) sendRoomState(ctx context.Context, connection netconn.Context, active *roomlive.Room, playerID int64) error {
	unitRecords := projection.Units(active, playerFilter(playerID)...)
	if len(unitRecords) > 0 {
		packet, err := outunits.Encode(unitRecords)
		if err != nil {
			return err
		}
		if err := connection.Send(ctx, packet); err != nil {
			return err
		}
	}

	statusRecords := projection.Statuses(active, playerFilter(playerID)...)
	if len(statusRecords) == 0 {
		return nil
	}
	packet, err := outstatus.Encode(statusRecords)
	if err != nil {
		return err
	}

	return connection.Send(ctx, packet)
}

// playerFilter returns an optional projection player filter.
func playerFilter(playerID int64) []int64 {
	if playerID <= 0 {
		return nil
	}

	return []int64{playerID}
}

// broadcastJoined sends the entered player unit to other room occupants.
func (handler Handler) broadcastJoined(ctx context.Context, active *roomlive.Room, playerID int64) error {
	if handler.Connections == nil {
		return nil
	}

	unitRecords := projection.Units(active, playerID)
	if len(unitRecords) > 0 {
		packet, err := outunits.Encode(unitRecords)
		if err != nil {
			return err
		}
		if err := broadcast.RoomPacket(ctx, handler.Connections, active, packet, playerID); err != nil {
			return err
		}
	}

	statusRecords := projection.Statuses(active, playerID)
	if len(statusRecords) == 0 {
		return nil
	}
	packet, err := outstatus.Encode(statusRecords)
	if err != nil {
		return err
	}

	return broadcast.RoomPacket(ctx, handler.Connections, active, packet, playerID)
}
