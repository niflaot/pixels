// Package presence projects player presence changes into friend-list updates.
package presence

import (
	"context"

	"github.com/niflaot/pixels/internal/realm/messenger/delivery"
	messengerservice "github.com/niflaot/pixels/internal/realm/messenger/service"
	outupdate "github.com/niflaot/pixels/networking/outbound/messenger/update"
)

// Broadcaster sends viewer-specific friend presence cards.
type Broadcaster struct {
	// messenger builds viewer-specific presence projections.
	messenger *messengerservice.Service
	// delivery sends packets only to online friends.
	delivery *delivery.Sender
}

// New creates messenger presence projection behavior.
func New(messenger *messengerservice.Service, delivery *delivery.Sender) *Broadcaster {
	return &Broadcaster{messenger: messenger, delivery: delivery}
}

// Update broadcasts one player's current online and room presence.
func (broadcaster *Broadcaster) Update(ctx context.Context, playerID int64) error {
	updates, err := broadcaster.messenger.PresenceUpdates(ctx, playerID, broadcaster.delivery.Online)
	if err != nil {
		return err
	}
	for _, update := range updates {
		packet, encodeErr := outupdate.Encode([]outupdate.Entry{{Type: outupdate.Changed, Card: delivery.FriendCard(update.Card)}})
		if encodeErr != nil {
			return encodeErr
		}
		if _, sendErr := broadcaster.delivery.Send(ctx, update.PlayerID, packet); sendErr != nil {
			return sendErr
		}
	}
	return nil
}
