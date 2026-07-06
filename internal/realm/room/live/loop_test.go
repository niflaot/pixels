package live

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestRoomLoopPublishesMovements verifies owner loop movement publishing.
func TestRoomLoopPublishesMovements(t *testing.T) {
	room := worldRoomForTest(t, "00", 0, 0)
	if _, err := room.Join(occupantForTest(7)); err != nil {
		t.Fatalf("join room: %v", err)
	}
	if _, err := room.MoveTo(7, pointForTest(t, 1, 0)); err != nil {
		t.Fatalf("move unit: %v", err)
	}

	var calls atomic.Int32
	room.startLoop(context.Background(), time.Millisecond, func(context.Context, *Room, []Movement) error {
		calls.Add(1)
		return nil
	})
	defer room.stopLoop()

	deadline := time.After(200 * time.Millisecond)
	for calls.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("expected movement publish")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

// TestRoomLoopIgnoresMissingPublisher verifies nil publishers do not start.
func TestRoomLoopIgnoresMissingPublisher(t *testing.T) {
	room := worldRoomForTest(t, "0", 0, 0)
	room.startLoop(context.Background(), time.Millisecond, nil)
	if room.loopCancel != nil {
		t.Fatal("expected no loop")
	}
}
