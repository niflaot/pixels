package live

import (
	"strconv"

	"github.com/niflaot/pixels/internal/realm/room/world/grid"
	worldpath "github.com/niflaot/pixels/internal/realm/room/world/path"
	"github.com/niflaot/pixels/internal/realm/room/world/surface"
	worldunit "github.com/niflaot/pixels/internal/realm/room/world/unit"
)

// MoveTo sets a unit movement goal.
func (room *Room) MoveTo(playerID int64, goal grid.Point) (worldpath.Path, error) {
	runtime, start, occupancy, err := room.movementSnapshot(playerID)
	if err != nil {
		return worldpath.Path{}, err
	}

	finder := worldpath.NewFinderWithOccupancy(runtime.resolver, runtime.rules, occupancy)
	roomPath, err := finder.Find(start, goal)
	if err != nil {
		return worldpath.Path{}, err
	}

	room.mutex.Lock()
	defer room.mutex.Unlock()

	if room.world != runtime {
		return worldpath.Path{}, worldpath.ErrInvalidPath
	}
	roomUnit, ok := room.world.units[playerID]
	if !ok {
		return worldpath.Path{}, ErrUnitNotFound
	}
	if err := roomPath.Validate(room.world.resolver); err != nil {
		return worldpath.Path{}, err
	}
	roomUnit.SetPath(roomPath)

	return roomPath, nil
}

// FaceTo rotates a unit toward a target point and clears pending movement.
func (room *Room) FaceTo(playerID int64, target grid.Point) (UnitSnapshot, error) {
	room.mutex.Lock()
	defer room.mutex.Unlock()

	if room.world == nil {
		return UnitSnapshot{}, ErrWorldNotLoaded
	}
	roomUnit, ok := room.world.units[playerID]
	if !ok {
		return UnitSnapshot{}, ErrUnitNotFound
	}
	roomUnit.ClearPath()
	roomUnit.FaceToward(target)

	return unitSnapshot(playerID, roomUnit), nil
}

// Tick advances room world movement once.
func (room *Room) Tick() []Movement {
	room.mutex.Lock()
	defer room.mutex.Unlock()

	if room.world == nil {
		return nil
	}

	playerIDs := room.world.sortedPlayerIDs()
	movements := make([]Movement, 0, len(playerIDs))
	for _, playerID := range playerIDs {
		roomUnit := room.world.units[playerID]
		if roomUnit.Moving() {
			if err := roomUnit.ValidatePath(room.world.resolver); err != nil {
				roomUnit.ClearPath()

				continue
			}
		}
		step, moved, settled := roomUnit.Advance()
		if !moved && !settled {
			continue
		}
		if settled {
			room.world.settleUnit(roomUnit)
		}
		movements = append(movements, Movement{
			PlayerID: playerID,
			Unit:     unitSnapshot(playerID, roomUnit),
			Step:     step,
			Moved:    moved,
			Settled:  settled,
		})
	}

	return movements
}

// movementSnapshot returns data needed to calculate movement outside the room lock.
func (room *Room) movementSnapshot(playerID int64) (*World, worldpath.Position, worldpath.Occupancy, error) {
	room.mutex.RLock()
	defer room.mutex.RUnlock()

	if room.world == nil {
		return nil, worldpath.Position{}, worldpath.Occupancy{}, ErrWorldNotLoaded
	}
	roomUnit, ok := room.world.units[playerID]
	if !ok {
		return nil, worldpath.Position{}, worldpath.Occupancy{}, ErrUnitNotFound
	}

	return room.world, roomUnit.Position(), room.world.occupancyExcept(playerID), nil
}

// settleUnit applies a sit or lay status when a unit lands on a seat or lay section.
func (world *World) settleUnit(roomUnit *worldunit.Unit) {
	position := roomUnit.Position()
	column, err := world.resolver.Column(position.Point)
	if err != nil {
		return
	}
	section, ok := column.SectionAt(position.Z)
	if !ok {
		return
	}

	switch section.State() {
	case surface.StateSit:
		roomUnit.Settle(worldunit.StatusSit, heightValue(section.Z()), roomUnit.BodyRotation(), roomUnit.HeadRotation())
	case surface.StateLay:
		roomUnit.Settle(worldunit.StatusLay, heightValue(section.Z()), roomUnit.BodyRotation(), roomUnit.HeadRotation())
	}
}

// heightValue formats a grid height for a unit status value.
func heightValue(height grid.Height) string {
	return strconv.Itoa(int(height))
}
