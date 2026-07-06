// Package walk contains the room walk packet handler.
package walk

import (
	"context"

	"github.com/niflaot/pixels/internal/command"
	walkcmd "github.com/niflaot/pixels/internal/realm/room/commands/walk"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	inwalk "github.com/niflaot/pixels/networking/inbound/room/entities/walk"
	"go.uber.org/zap"
)

// New creates a room walk packet handler.
func New(handler walkcmd.Handler, log *zap.Logger) netconn.Handler {
	dispatcher, _ := command.NewDispatcher(handler)
	dispatcher.WithLogger(log)

	return func(connection netconn.Context, packet codec.Packet) error {
		payload, err := inwalk.Decode(packet)
		if err != nil {
			return err
		}

		return dispatcher.Dispatch(context.Background(), command.Envelope[walkcmd.Command]{
			Command:  walkcmd.Command{Handler: connection, X: int(payload.X), Y: int(payload.Y)},
			Metadata: command.Metadata{ConnectionID: string(connection.ConnectionID)},
		})
	}
}

// Register adds the room walk handler to a registry.
func Register(registry *netconn.HandlerRegistry, handler netconn.Handler) {
	_ = registry.Register(inwalk.Header, handler)
}
