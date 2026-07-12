package privacy

import (
	"context"

	"github.com/niflaot/pixels/internal/command"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	inignoreid "github.com/niflaot/pixels/networking/inbound/user/ignore/id"
	inignorelist "github.com/niflaot/pixels/networking/inbound/user/ignore/list"
	inignorename "github.com/niflaot/pixels/networking/inbound/user/ignore/name"
	inignoreremove "github.com/niflaot/pixels/networking/inbound/user/ignore/remove"
	inrelationships "github.com/niflaot/pixels/networking/inbound/user/relationships"
	"go.uber.org/zap"
)

// RegisterHandlers registers ignored-user and relationship packet adapters.
func RegisterHandlers(registry *netconn.HandlerRegistry, handler Handler, log *zap.Logger) {
	register(registry, inignorelist.Header, func(packet codec.Packet) (Command, error) {
		_, err := inignorelist.Decode(packet)
		return Command{Name: ListName}, err
	}, handler, log)
	register(registry, inignorename.Header, func(packet codec.Packet) (Command, error) {
		username, err := inignorename.Decode(packet)
		return Command{Name: IgnoreName, Username: username}, err
	}, handler, log)
	register(registry, inignoreid.Header, func(packet codec.Packet) (Command, error) {
		playerID, err := inignoreid.Decode(packet)
		return Command{Name: IgnoreName, PlayerID: playerID}, err
	}, handler, log)
	register(registry, inignoreremove.Header, func(packet codec.Packet) (Command, error) {
		username, err := inignoreremove.Decode(packet)
		return Command{Name: UnignoreName, Username: username}, err
	}, handler, log)
	register(registry, inrelationships.Header, func(packet codec.Packet) (Command, error) {
		playerID, err := inrelationships.Decode(packet)
		return Command{Name: RelationshipsName, PlayerID: playerID}, err
	}, handler, log)
}

// commandHandler adapts privacy commands to domain behavior.
type commandHandler struct {
	// handler stores privacy behavior.
	handler Handler
}

// Handle executes one privacy command envelope.
func (handler commandHandler) Handle(ctx context.Context, envelope command.Envelope[Command]) error {
	return handler.handler.Handle(ctx, envelope.Command)
}

// decoder decodes one privacy packet.
type decoder func(codec.Packet) (Command, error)

// register installs one privacy command adapter.
func register(registry *netconn.HandlerRegistry, header uint16, decode decoder, handler Handler, log *zap.Logger) {
	dispatcher, _ := command.NewDispatcher(commandHandler{handler: handler})
	dispatcher.WithLogger(log)
	_ = registry.Register(header, func(connection netconn.Context, packet codec.Packet) error {
		input, err := decode(packet)
		if err != nil {
			return err
		}
		input.Connection = connection
		return dispatcher.Dispatch(context.Background(), command.Envelope[Command]{Command: input, Metadata: command.Metadata{ConnectionID: string(connection.ConnectionID)}})
	})
}
