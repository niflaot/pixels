// Package broadcast sends room runtime packets to active connections.
package broadcast

import (
	"context"
	"errors"

	"github.com/niflaot/pixels/internal/realm/room/live"
	"github.com/niflaot/pixels/internal/realm/room/projection"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	outremoved "github.com/niflaot/pixels/networking/outbound/room/entities/removed"
	outstatus "github.com/niflaot/pixels/networking/outbound/room/entities/status"
	outunits "github.com/niflaot/pixels/networking/outbound/room/entities/units"
)

// NewMovementPublisher creates a movement broadcaster.
func NewMovementPublisher(connections *netconn.Registry) live.MovementPublisher {
	return func(ctx context.Context, active *live.Room, movements []live.Movement) error {
		if connections == nil || active == nil || len(movements) == 0 {
			return nil
		}

		packet, err := outstatus.Encode(projection.MovementStatuses(movements))
		if err != nil {
			return err
		}

		return RoomPacket(ctx, connections, active, packet, 0)
	}
}

// RoomPacket sends a packet to active room occupants.
func RoomPacket(ctx context.Context, connections *netconn.Registry, active *live.Room, packet codec.Packet, excludedPlayerID int64) error {
	if connections == nil || active == nil {
		return nil
	}

	var sendErr error
	for _, occupant := range active.Occupants() {
		if occupant.PlayerID == excludedPlayerID {
			continue
		}
		connection, found := connections.Get(occupant.ConnectionKind, occupant.ConnectionID)
		if !found {
			continue
		}
		if err := connection.Send(ctx, packet); err != nil {
			sendErr = errors.Join(sendErr, err)
		}
	}

	return sendErr
}

// RoomSpawn sends a unit spawn and initial status to active room occupants.
func RoomSpawn(ctx context.Context, connections *netconn.Registry, active *live.Room, playerID int64, excludedPlayerID int64) error {
	unitRecords := projection.Units(active, playerID)
	if len(unitRecords) > 0 {
		packet, err := outunits.Encode(unitRecords)
		if err != nil {
			return err
		}
		if err := RoomPacket(ctx, connections, active, packet, excludedPlayerID); err != nil {
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

	return RoomPacket(ctx, connections, active, packet, excludedPlayerID)
}

// RoomRemove sends a room unit remove packet to room occupants.
func RoomRemove(ctx context.Context, connections *netconn.Registry, active *live.Room, unitID int64, excludedPlayerID int64) error {
	packet, err := outremoved.Encode(unitID)
	if err != nil {
		return err
	}

	return RoomPacket(ctx, connections, active, packet, excludedPlayerID)
}
