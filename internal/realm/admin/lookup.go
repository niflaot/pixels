package admin

import (
	"strings"

	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	"github.com/niflaot/pixels/internal/realm/session/binding"
	netconn "github.com/niflaot/pixels/networking/connection"
)

// findPlayer resolves one connected player by exact case-insensitive username.
func (service *Service) findPlayer(username string) (*playerlive.Player, bool) {
	for _, player := range service.players.Snapshot() {
		if strings.EqualFold(player.Username(), strings.TrimSpace(username)) {
			return player, true
		}
	}

	return nil, false
}

// connection resolves one player's current transport.
func (service *Service) connection(playerID int64) (netconn.Connection, error) {
	current, found := service.bindings.FindByPlayer(playerID)
	if !found {
		return nil, binding.ErrBindingNotFound
	}
	connection, found := service.connections.Get(current.ConnectionKind, current.ConnectionID)
	if !found {
		return nil, netconn.ErrConnectionNotFound
	}

	return connection, nil
}
