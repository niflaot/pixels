// Package packet contains reusable pixel-protocol packet primitives.
package packet

// Packet is a decoded pixel-protocol message.
type Packet struct {
	Header uint16
	Body   []byte
}

// New creates a packet with a defensive body copy.
func New(header uint16, body []byte) Packet {
	copied := append([]byte(nil), body...)

	return Packet{
		Header: header,
		Body:   copied,
	}
}

// Empty reports whether the packet has no body bytes.
func (packet Packet) Empty() bool {
	return len(packet.Body) == 0
}
