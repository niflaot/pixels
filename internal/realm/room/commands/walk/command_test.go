package walk

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/niflaot/pixels/internal/command"
	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	roomlive "github.com/niflaot/pixels/internal/realm/room/live"
	"github.com/niflaot/pixels/internal/realm/room/world/grid"
	worldpath "github.com/niflaot/pixels/internal/realm/room/world/path"
	worldunit "github.com/niflaot/pixels/internal/realm/room/world/unit"
	"github.com/niflaot/pixels/internal/realm/session/binding"
	netconn "github.com/niflaot/pixels/networking/connection"
)

// TestHandleMovesPlayer verifies walk command handling.
func TestHandleMovesPlayer(t *testing.T) {
	handler, player := handlerForTest(t)
	if err := player.EnterRoom(9); err != nil {
		t.Fatalf("enter room: %v", err)
	}

	err := handler.Handle(context.Background(), command.Envelope[Command]{
		Command: Command{Handler: connectionForTest(), X: 1, Y: 0},
	})
	if err != nil {
		t.Fatalf("handle walk: %v", err)
	}
	room, _ := handler.Runtime.Find(9)
	if units := room.Units(); len(units) != 1 || !units[0].Moving {
		t.Fatalf("expected moving unit %#v", units)
	}
}

// TestHandleRejectsMissingRoomPresence verifies room presence validation.
func TestHandleRejectsMissingRoomPresence(t *testing.T) {
	handler, _ := handlerForTest(t)
	err := handler.Handle(context.Background(), command.Envelope[Command]{
		Command: Command{Handler: connectionForTest(), X: 1, Y: 0},
	})
	if !errors.Is(err, ErrPlayerNotInRoom) {
		t.Fatalf("expected player not in room, got %v", err)
	}
}

// TestHandleRejectsInvalidTarget verifies target validation.
func TestHandleRejectsInvalidTarget(t *testing.T) {
	handler, player := handlerForTest(t)
	if err := player.EnterRoom(9); err != nil {
		t.Fatalf("enter room: %v", err)
	}

	err := handler.Handle(context.Background(), command.Envelope[Command]{
		Command: Command{Handler: connectionForTest(), X: -1, Y: 0},
	})
	if !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("expected invalid target, got %v", err)
	}
}

// handlerForTest creates a walk command handler.
func handlerForTest(t *testing.T) (Handler, *playerlive.Player) {
	t.Helper()

	players := playerlive.NewRegistry()
	peer, err := playerlive.NewSessionPeer(netconn.ID("conn"), netconn.Kind("websocket"), time.Now())
	if err != nil {
		t.Fatalf("create peer: %v", err)
	}
	player, err := playerlive.NewPlayer(playerlive.Snapshot{ID: 7, Username: "demo"}, peer)
	if err != nil {
		t.Fatalf("create player: %v", err)
	}
	if err := players.Add(player); err != nil {
		t.Fatalf("add player: %v", err)
	}

	bindings := binding.NewRegistry()
	if err := bindings.Add(binding.Binding{PlayerID: 7, ConnectionID: netconn.ID("conn"), ConnectionKind: netconn.Kind("websocket")}); err != nil {
		t.Fatalf("add binding: %v", err)
	}

	runtime := roomlive.NewRegistry(nil)
	room, err := runtime.Activate(roomlive.Snapshot{ID: 9, MaxUsers: 10})
	if err != nil {
		t.Fatalf("activate room: %v", err)
	}
	roomGrid, err := grid.Parse("00", grid.WithDoor(0, 0))
	if err != nil {
		t.Fatalf("parse grid: %v", err)
	}
	err = room.LoadWorld(roomlive.WorldConfig{
		Grid: roomGrid,
		Door: worldpath.Position{Point: grid.MustPoint(0, 0)},
		Body: worldunit.RotationSouth,
		Head: worldunit.RotationSouth,
	})
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	if _, err := runtime.Join(context.Background(), 9, occupantForTest(7)); err != nil {
		t.Fatalf("join runtime: %v", err)
	}

	return Handler{Players: players, Bindings: bindings, Runtime: runtime}, player
}

// connectionForTest creates a connection context.
func connectionForTest() netconn.Context {
	return netconn.Context{ConnectionID: netconn.ID("conn"), ConnectionKind: netconn.Kind("websocket")}
}

// occupantForTest creates a room occupant.
func occupantForTest(playerID int64) roomlive.Occupant {
	return roomlive.Occupant{PlayerID: playerID, Username: "demo", ConnectionID: netconn.ID("conn"), ConnectionKind: netconn.Kind("websocket")}
}
