package live

import (
	"errors"
	"testing"

	worldpath "github.com/niflaot/pixels/internal/realm/room/world/path"
	worldunit "github.com/niflaot/pixels/internal/realm/room/world/unit"
)

// TestRoomMoveToAndTickAdvancesUnit verifies runtime movement ticks.
func TestRoomMoveToAndTickAdvancesUnit(t *testing.T) {
	room := worldRoomForTest(t, "000", 0, 0)
	if _, err := room.Join(occupantForTest(7)); err != nil {
		t.Fatalf("join room: %v", err)
	}

	path, err := room.MoveTo(7, pointForTest(t, 2, 0))
	if err != nil {
		t.Fatalf("move unit: %v", err)
	}
	if path.Len() != 2 {
		t.Fatalf("unexpected path length %d", path.Len())
	}

	first := room.Tick()
	if len(first) != 1 || first[0].Unit.Position.Point != pointForTest(t, 1, 0) {
		t.Fatalf("unexpected first tick %#v", first)
	}
	if !hasStatus(first[0].Unit.Statuses, worldunit.StatusMove) {
		t.Fatalf("expected move status %#v", first[0].Unit.Statuses)
	}

	second := room.Tick()
	if len(second) != 1 || second[0].Unit.Position.Point != pointForTest(t, 2, 0) {
		t.Fatalf("unexpected second tick %#v", second)
	}
	if second[0].Unit.Moving || !second[0].Moved || second[0].Settled {
		t.Fatalf("expected movement completed %#v", second[0].Unit)
	}
	if !hasStatus(second[0].Unit.Statuses, worldunit.StatusMove) {
		t.Fatalf("expected final move status %#v", second[0].Unit.Statuses)
	}

	third := room.Tick()
	if len(third) != 1 || third[0].Moved || !third[0].Settled {
		t.Fatalf("unexpected settle tick %#v", third)
	}
	if hasStatus(third[0].Unit.Statuses, worldunit.StatusMove) {
		t.Fatalf("expected clean settled status %#v", third[0].Unit.Statuses)
	}
}

// TestRoomFaceToClearsMovementAndRotates verifies facing an occupied target.
func TestRoomFaceToClearsMovementAndRotates(t *testing.T) {
	room := worldRoomForTest(t, "00", 0, 0)
	if _, err := room.Join(occupantForTest(7)); err != nil {
		t.Fatalf("join room: %v", err)
	}
	if _, err := room.MoveTo(7, pointForTest(t, 1, 0)); err != nil {
		t.Fatalf("move unit: %v", err)
	}

	unit, err := room.FaceTo(7, pointForTest(t, 1, 0))
	if err != nil {
		t.Fatalf("face unit: %v", err)
	}
	if unit.Moving || unit.BodyRotation != worldunit.RotationEast || unit.HeadRotation != worldunit.RotationEast {
		t.Fatalf("unexpected faced unit %#v", unit)
	}
}

// TestRoomMoveToAvoidsOccupiedUnit verifies occupancy-aware paths.
func TestRoomMoveToAvoidsOccupiedUnit(t *testing.T) {
	room := worldRoomForTest(t, "000\r000\r000", 0, 1)
	if _, err := room.Join(occupantForTest(7)); err != nil {
		t.Fatalf("join first room unit: %v", err)
	}
	if _, err := room.Join(occupantForTest(8)); err != nil {
		t.Fatalf("join second room unit: %v", err)
	}
	if _, err := room.MoveTo(8, pointForTest(t, 1, 1)); err != nil {
		t.Fatalf("move blocker: %v", err)
	}
	if movements := room.Tick(); len(movements) != 1 {
		t.Fatalf("expected blocker movement %#v", movements)
	}

	path, err := room.MoveTo(7, pointForTest(t, 2, 1))
	if err != nil {
		t.Fatalf("move around blocker: %v", err)
	}
	for _, step := range path.Steps() {
		if step.Position.Point == pointForTest(t, 1, 1) {
			t.Fatalf("path stepped into occupied tile %#v", path.Steps())
		}
	}
}

// TestRoomMoveToAvoidsReservedGoal verifies pending movement targets are blocked.
func TestRoomMoveToAvoidsReservedGoal(t *testing.T) {
	room := worldRoomForTest(t, "000", 0, 0)
	if _, err := room.Join(occupantForTest(7)); err != nil {
		t.Fatalf("join first room unit: %v", err)
	}
	if _, err := room.Join(occupantForTest(8)); err != nil {
		t.Fatalf("join second room unit: %v", err)
	}
	if _, err := room.MoveTo(8, pointForTest(t, 1, 0)); err != nil {
		t.Fatalf("move second room unit: %v", err)
	}
	if movements := room.Tick(); len(movements) != 1 {
		t.Fatalf("expected second movement %#v", movements)
	}
	if _, err := room.MoveTo(8, pointForTest(t, 2, 0)); err != nil {
		t.Fatalf("reserve second goal: %v", err)
	}

	_, err := room.MoveTo(7, pointForTest(t, 2, 0))
	if !errors.Is(err, worldpath.ErrNoPath) {
		t.Fatalf("expected reserved goal to block path, got %v", err)
	}
}
