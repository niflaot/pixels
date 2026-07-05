package connection

import "github.com/niflaot/pixels/networking/codec"

// Direction names whether a packet is entering or leaving a connection.
type Direction uint8

const (
	// InboundDirection describes a packet received from the peer.
	InboundDirection Direction = iota + 1

	// OutboundDirection describes a packet sent to the peer.
	OutboundDirection
)

// Command is emitted by handlers for the realm layer to execute.
type Command struct {
	// Name identifies the command for the realm layer.
	Name string
	// ConnectionID is the connection that emitted the command.
	ConnectionID ID
	// ConnectionKind is the kind of connection that emitted the command.
	ConnectionKind Kind
	// Direction is the packet flow direction that emitted the command.
	Direction Direction
	// Packet is the packet that caused the command.
	Packet codec.Packet
	// Data stores optional decoded handler data.
	Data any
}

// NewCommand creates a command for a packet handling event.
func NewCommand(name string, context Context, packet codec.Packet, data any) Command {
	return Command{
		Name:           name,
		ConnectionID:   context.ConnectionID,
		ConnectionKind: context.ConnectionKind,
		Direction:      context.Direction,
		Packet:         packet,
		Data:           data,
	}
}
