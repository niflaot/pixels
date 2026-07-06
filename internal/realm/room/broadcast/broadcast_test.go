package broadcast

import (
	"context"
	"errors"
	"testing"

	"github.com/niflaot/pixels/internal/realm/room/live"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
)

// TestRoomPacketSendsToOccupants verifies room broadcast delivery.
func TestRoomPacketSendsToOccupants(t *testing.T) {
	connections := netconn.NewRegistry()
	sent := registerConnectionForTest(t, connections, "conn")
	room, err := live.NewRoom(live.Snapshot{ID: 9, MaxUsers: 5})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := room.Join(live.Occupant{PlayerID: 7, Username: "demo", ConnectionID: "conn", ConnectionKind: "websocket"}); err != nil {
		t.Fatalf("join room: %v", err)
	}

	err = RoomPacket(context.Background(), connections, room, codec.Packet{Header: 9}, 0)
	if err != nil {
		t.Fatalf("broadcast packet: %v", err)
	}
	if len(*sent) != 1 || (*sent)[0].Header != 9 {
		t.Fatalf("unexpected sent packets %#v", *sent)
	}

	err = RoomPacket(context.Background(), connections, room, codec.Packet{Header: 10}, 7)
	if err != nil {
		t.Fatalf("broadcast excluded packet: %v", err)
	}
	if len(*sent) != 1 {
		t.Fatalf("expected excluded packet to be skipped %#v", *sent)
	}
}

// TestRoomRemoveEncodesPacket verifies remove broadcasting.
func TestRoomRemoveEncodesPacket(t *testing.T) {
	connections := netconn.NewRegistry()
	sent := registerConnectionForTest(t, connections, "conn")
	room, err := live.NewRoom(live.Snapshot{ID: 9, MaxUsers: 5})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := room.Join(live.Occupant{PlayerID: 7, Username: "demo", ConnectionID: "conn", ConnectionKind: "websocket"}); err != nil {
		t.Fatalf("join room: %v", err)
	}

	err = RoomRemove(context.Background(), connections, room, 7, 0)
	if err != nil {
		t.Fatalf("remove packet: %v", err)
	}
	if len(*sent) != 1 || (*sent)[0].Header != 2661 {
		t.Fatalf("unexpected sent packets %#v", *sent)
	}
}

// TestMovementPublisherSendsStatus verifies movement publisher wiring.
func TestMovementPublisherSendsStatus(t *testing.T) {
	connections := netconn.NewRegistry()
	sent := registerConnectionForTest(t, connections, "conn")
	room, err := live.NewRoom(live.Snapshot{ID: 9, MaxUsers: 5})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := room.Join(live.Occupant{PlayerID: 7, Username: "demo", ConnectionID: "conn", ConnectionKind: "websocket"}); err != nil {
		t.Fatalf("join room: %v", err)
	}

	publisher := NewMovementPublisher(connections)
	err = publisher(context.Background(), room, []live.Movement{{Unit: live.UnitSnapshot{UnitID: 1}}})
	if err != nil {
		t.Fatalf("publish movement: %v", err)
	}
	if len(*sent) != 1 || (*sent)[0].Header != 1640 {
		t.Fatalf("unexpected sent packets %#v", *sent)
	}
}

// TestMovementPublisherSkipsMissingState verifies movement no-op guards.
func TestMovementPublisherSkipsMissingState(t *testing.T) {
	publisher := NewMovementPublisher(nil)
	if err := publisher(context.Background(), nil, nil); err != nil {
		t.Fatalf("publish empty movement: %v", err)
	}
}

// TestRoomPacketHandlesMissingConnection verifies stale occupant connections.
func TestRoomPacketHandlesMissingConnection(t *testing.T) {
	room, err := live.NewRoom(live.Snapshot{ID: 9, MaxUsers: 5})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := room.Join(live.Occupant{PlayerID: 7, Username: "demo", ConnectionID: "missing", ConnectionKind: "websocket"}); err != nil {
		t.Fatalf("join room: %v", err)
	}

	err = RoomPacket(context.Background(), netconn.NewRegistry(), room, codec.Packet{Header: 9}, 0)
	if err != nil {
		t.Fatalf("broadcast missing connection: %v", err)
	}
}

// TestRoomPacketReturnsSendError verifies send failures propagate.
func TestRoomPacketReturnsSendError(t *testing.T) {
	sendErr := errors.New("send failed")
	connections := netconn.NewRegistry()
	outbound := netconn.NewHandlerRegistry()
	outbound.SetFallback(func(netconn.Context, codec.Packet) error {
		return nil
	}, netconn.AllowAnyActiveState(), netconn.AllowUnauthenticated())
	session, err := netconn.NewSession(netconn.SessionConfig{
		ID:       "conn",
		Kind:     "websocket",
		Outbound: outbound,
		Sender: func(context.Context, codec.Packet) error {
			return sendErr
		},
		Disposer: func(context.Context, netconn.Reason) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := connections.Register(session); err != nil {
		t.Fatalf("register session: %v", err)
	}
	room, err := live.NewRoom(live.Snapshot{ID: 9, MaxUsers: 5})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := room.Join(live.Occupant{PlayerID: 7, Username: "demo", ConnectionID: "conn", ConnectionKind: "websocket"}); err != nil {
		t.Fatalf("join room: %v", err)
	}

	err = RoomPacket(context.Background(), connections, room, codec.Packet{Header: 9}, 0)
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected send error, got %v", err)
	}
}

// registerConnectionForTest registers a captured test connection.
func registerConnectionForTest(t *testing.T, connections *netconn.Registry, id netconn.ID) *[]codec.Packet {
	t.Helper()

	sent := make([]codec.Packet, 0, 1)
	outbound := netconn.NewHandlerRegistry()
	outbound.SetFallback(func(netconn.Context, codec.Packet) error {
		return nil
	}, netconn.AllowAnyActiveState(), netconn.AllowUnauthenticated())
	session, err := netconn.NewSession(netconn.SessionConfig{
		ID:       id,
		Kind:     "websocket",
		Outbound: outbound,
		Sender: func(_ context.Context, packet codec.Packet) error {
			sent = append(sent, packet)
			return nil
		},
		Disposer: func(context.Context, netconn.Reason) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := connections.Register(session); err != nil {
		t.Fatalf("register session: %v", err)
	}

	return &sent
}
