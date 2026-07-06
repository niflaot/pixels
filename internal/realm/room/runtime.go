package room

import (
	"context"

	playerdisconnected "github.com/niflaot/pixels/internal/realm/player/events/disconnected"
	roomoccupancy "github.com/niflaot/pixels/internal/realm/room/events/occupancychanged"
	"github.com/niflaot/pixels/internal/realm/room/live"
	"github.com/niflaot/pixels/pkg/bus"
	"go.uber.org/fx"
)

// NewLiveRegistry creates the active room registry.
func NewLiveRegistry(publisher bus.Publisher) *live.Registry {
	return live.NewRegistry(func(ctx context.Context, occupancy live.Occupancy) error {
		return publisher.Publish(ctx, bus.Event{Name: roomoccupancy.Name, Payload: occupancyEvent(occupancy)})
	})
}

// RegisterRuntimeCleanup removes room occupancy on player disconnect.
func RegisterRuntimeCleanup(lifecycle fx.Lifecycle, subscriber bus.Subscriber, registry *live.Registry) error {
	subscription, err := subscriber.Subscribe(playerdisconnected.Name, bus.PriorityNormal, func(ctx context.Context, event bus.Event) error {
		disconnected, ok := event.Payload.(playerdisconnected.Payload)
		if !ok || disconnected.PlayerID <= 0 {
			return nil
		}

		_, _, removeErr := registry.RemovePlayer(ctx, disconnected.PlayerID)

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
