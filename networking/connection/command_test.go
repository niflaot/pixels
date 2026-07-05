package connection

import (
	"testing"

	"github.com/niflaot/pixels/networking/codec"
)

// TestNewCommandCopiesContext verifies command metadata creation.
func TestNewCommandCopiesContext(t *testing.T) {
	context := Context{
		ConnectionID:   "one",
		ConnectionKind: "websocket",
		Direction:      InboundDirection,
	}
	packet := codec.Packet{Header: 7}
	command := NewCommand("authenticate", context, packet, "payload")

	if command.Name != "authenticate" {
		t.Fatalf("expected command name, got %s", command.Name)
	}

	if command.ConnectionID != context.ConnectionID {
		t.Fatalf("expected connection id %s, got %s", context.ConnectionID, command.ConnectionID)
	}

	if command.Packet.Header != packet.Header {
		t.Fatalf("expected packet header %d, got %d", packet.Header, command.Packet.Header)
	}
}
