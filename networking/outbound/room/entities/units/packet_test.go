package units

import (
	"testing"

	"github.com/niflaot/pixels/networking/codec"
)

// TestEncode verifies UNIT packet encoding.
func TestEncode(t *testing.T) {
	packet, err := Encode([]Unit{{
		UserID: 7, Name: "demo", Motto: "hi", Figure: "hd-180-1",
		RoomIndex: 3, X: 1, Y: 2, Z: "0", Direction: 4,
		Gender: "M", GroupID: -1, GroupStatus: -1, Moderator: true,
	}})
	if err != nil {
		t.Fatalf("encode packet: %v", err)
	}
	if packet.Header != Header {
		t.Fatalf("unexpected header %d", packet.Header)
	}

	values, rest, err := codec.DecodePayload(nil, codec.Definition{codec.Int32Field}, packet.Payload)
	if err != nil {
		t.Fatalf("decode count: %v", err)
	}
	if values[0].Int32 != 1 || len(rest) == 0 {
		t.Fatalf("unexpected count=%#v rest=%d", values, len(rest))
	}
}
