package runtime

import (
	worldfurniture "github.com/niflaot/pixels/internal/realm/room/world/furniture"
	"github.com/niflaot/pixels/internal/realm/room/world/grid"
	"github.com/niflaot/pixels/internal/realm/room/world/surface"
	worldunit "github.com/niflaot/pixels/internal/realm/room/world/unit"
)

// ResolveFurniturePlacement validates occupancy and stacking for a footprint.
func (world *World) ResolveFurniturePlacement(sourceID int64, footprint []grid.Point) (grid.Height, error) {
	occupied := world.unitPositionsExcludingSource(sourceID)
	var height grid.Height
	for _, point := range footprint {
		if _, blocked := occupied[point]; blocked {
			return 0, ErrTileOccupied
		}
		baseHeight, validTile := world.grid.HeightAt(point)
		if !validTile {
			return 0, ErrInvalidPlacement
		}
		column, err := world.resolver.Column(point)
		if err != nil {
			return 0, ErrInvalidPlacement
		}
		top, ok := topExcludingSource(column, sourceID)
		if !ok {
			if baseHeight > height {
				height = baseHeight
			}
			continue
		}
		if !top.Stacking() {
			return 0, ErrCannotStack
		}
		if top.Top() > height {
			height = top.Top()
		}
	}

	return height, nil
}

// unitPositionsExcludingSource returns occupied tiles except the source item's seated unit.
func (world *World) unitPositionsExcludingSource(sourceID int64) map[grid.Point]struct{} {
	ownSlots := make(map[grid.Point]struct{})
	if item, ok := world.furniture[sourceID]; ok {
		for _, slot := range worldfurniture.Slots(item) {
			ownSlots[slot.Point] = struct{}{}
		}
	}
	occupied := make(map[grid.Point]struct{}, len(world.units))
	for playerID, roomUnit := range world.units {
		point := roomUnit.Position().Point
		if _, isOwnSlot := ownSlots[point]; isOwnSlot {
			if occupantID, sat := world.slotOccupants[point]; sat && occupantID == playerID {
				continue
			}
		}
		occupied[point] = struct{}{}
	}

	return occupied
}

// topExcludingSource returns the highest section not owned by sourceID.
func topExcludingSource(column surface.Column, sourceID int64) (surface.Section, bool) {
	sections := column.Sections()
	for index := len(sections) - 1; index >= 0; index-- {
		if sections[index].SourceID() != sourceID {
			return sections[index], true
		}
	}

	return surface.Section{}, false
}

// reconcileSlotOccupants updates units affected by one changed furniture item.
func (world *World) reconcileSlotOccupants(previousSlots []worldfurniture.Slot, item *worldfurniture.Item) []UnitSnapshot {
	var updatedSlots []worldfurniture.Slot
	if item != nil {
		updatedSlots = worldfurniture.Slots(*item)
	}
	var affected []UnitSnapshot
	for _, previousSlot := range previousSlots {
		playerID, occupied := world.slotOccupants[previousSlot.Point]
		if !occupied {
			continue
		}
		roomUnit, ok := world.units[playerID]
		if !ok {
			continue
		}
		updatedSlot, found := slotAtPoint(updatedSlots, previousSlot.Point)
		if !found {
			world.releaseSlot(playerID)
			roomUnit.StandUp()
			if section, err := world.resolver.TopSection(previousSlot.Point); err == nil {
				roomUnit.SetHeight(section.Z())
			}
			affected = append(affected, unitSnapshot(playerID, roomUnit))
			continue
		}
		roomUnit.Settle(unitStatusFor(updatedSlot.Status), heightValue(updatedSlot.Z-item.Z), updatedSlot.BodyRotation, updatedSlot.BodyRotation)
		affected = append(affected, unitSnapshot(playerID, roomUnit))
	}

	return affected
}

// occupySlot records a player's current furniture slot.
func (world *World) occupySlot(playerID int64, point grid.Point) {
	world.releaseSlot(playerID)
	world.unitSlots[playerID] = point
	world.slotOccupants[point] = playerID
}

// releaseSlot removes a player's furniture slot reservation.
func (world *World) releaseSlot(playerID int64) {
	point, ok := world.unitSlots[playerID]
	if !ok {
		return
	}
	delete(world.unitSlots, playerID)
	delete(world.slotOccupants, point)
}

// slotAtPoint finds a slot at an exact point.
func slotAtPoint(slots []worldfurniture.Slot, point grid.Point) (worldfurniture.Slot, bool) {
	for _, slot := range slots {
		if slot.Point == point {
			return slot, true
		}
	}

	return worldfurniture.Slot{}, false
}

// unitStatusFor maps a furniture slot status to a unit status.
func unitStatusFor(status worldfurniture.SlotStatus) string {
	if status == worldfurniture.SlotStatusLay {
		return worldunit.StatusLay
	}

	return worldunit.StatusSit
}
