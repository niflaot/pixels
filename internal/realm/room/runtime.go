package room

import (
	"context"

	playerdisconnected "github.com/niflaot/pixels/internal/realm/player/events/disconnected"
	"github.com/niflaot/pixels/internal/realm/room/broadcast"
	roomoccupancy "github.com/niflaot/pixels/internal/realm/room/events/occupancychanged"
	"github.com/niflaot/pixels/internal/realm/room/live"
	netconn "github.com/niflaot/pixels/networking/connection"
	"github.com/niflaot/pixels/pkg/bus"
	"go.uber.org/fx"
)

// NewLiveRegistry creates the active room registry.
func NewLiveRegistry(publisher bus.Publisher, connections *netconn.Registry) *live.Registry {
	return live.NewRegistry(func(ctx context.Context, occupancy live.Occupancy) error {
		return publisher.Publish(ctx, bus.Event{Name: roomoccupancy.Name, Payload: occupancyEvent(occupancy)})
	}, live.WithMovementPublisher(broadcast.NewMovementPublisher(connections)))
}

// RegisterRuntimeCleanup removes room occupancy on player disconnect.
func RegisterRuntimeCleanup(lifecycle fx.Lifecycle, subscriber bus.Subscriber, registry *live.Registry, connections *netconn.Registry) error {
	subscription, err := subscriber.Subscribe(playerdisconnected.Name, bus.PriorityNormal, func(ctx context.Context, event bus.Event) error {
		disconnected, ok := event.Payload.(playerdisconnected.Payload)
		if !ok || disconnected.PlayerID <= 0 {
			return nil
		}

		room, found := registry.FindByPlayer(disconnected.PlayerID)
		unitID := unitIDForPlayer(room, disconnected.PlayerID)
		_, _, removeErr := registry.RemovePlayer(ctx, disconnected.PlayerID)
		if removeErr == nil && found && unitID > 0 {
			_ = broadcast.RoomRemove(ctx, connections, room, unitID, disconnected.PlayerID)
		}

		return removeErr
	})
	if err != nil {
		return err
	}

	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		subscription.Unsubscribe()
		return nil
	}})

	return nil
}

// unitIDForPlayer returns the live unit id for a player.
func unitIDForPlayer(room *live.Room, playerID int64) int64 {
	if room == nil {
		return 0
	}
	for _, unit := range room.Units() {
		if unit.PlayerID == playerID {
			return unit.UnitID
		}
	}

	return 0
}

// occupancyEvent maps live occupancy to a realm event payload.
func occupancyEvent(occupancy live.Occupancy) roomoccupancy.Payload {
	return roomoccupancy.Payload{
		RoomID:     occupancy.RoomID,
		CategoryID: occupancy.CategoryID,
		Count:      occupancy.Count,
		MaxUsers:   occupancy.MaxUsers,
		PlayerIDs:  append([]int64(nil), occupancy.PlayerIDs...),
	}
}
