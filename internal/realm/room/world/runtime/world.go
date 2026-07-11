package runtime

import (
	"fmt"

	worldfurniture "github.com/niflaot/pixels/internal/realm/room/world/furniture"
	"github.com/niflaot/pixels/internal/realm/room/world/grid"
	worldpath "github.com/niflaot/pixels/internal/realm/room/world/path"
	"github.com/niflaot/pixels/internal/realm/room/world/surface"
	worldunit "github.com/niflaot/pixels/internal/realm/room/world/unit"
)

// World stores loaded room world state.
type World struct {
	// grid stores the immutable world grid.
	grid grid.Grid
	// resolver resolves world columns.
	resolver *surface.Resolver
	// furniture stores placed furniture items by id.
	furniture map[int64]worldfurniture.Item
	// door stores the room entry position.
	door worldpath.Position
	// body stores the initial body rotation.
	body worldunit.Rotation
	// head stores the initial head rotation.
	head worldunit.Rotation
	// rules stores movement pathfinding rules.
	rules worldpath.Rules
	// units stores world units by player id.
	units map[int64]*worldunit.Unit
	// nextUnitID stores the next room-local unit id.
	nextUnitID int64
	// unitSlots stores the slot tile occupied by each player.
	unitSlots map[int64]grid.Point
	// slotOccupants stores the player occupying each slot tile.
	slotOccupants map[grid.Point]int64
}

// New creates loaded world state.
func New(config Config) (*World, error) {
	fixtures, furnitureIndex, err := furnitureFixtures(config.Furniture)
	if err != nil {
		return nil, err
	}
	fixtures = append(fixtures, config.Fixtures...)
	resolver, err := surface.NewResolver(config.Grid, fixtures)
	if err != nil {
		return nil, err
	}
	if _, err := resolver.SectionAt(config.Door.Point, config.Door.Z); err != nil {
		return nil, ErrInvalidWorld
	}

	return &World{
		grid: config.Grid, resolver: resolver, furniture: furnitureIndex,
		door: config.Door, body: config.Body, head: config.Head, rules: config.Rules.Normalize(),
		units: make(map[int64]*worldunit.Unit), nextUnitID: 1,
		unitSlots: make(map[int64]grid.Point), slotOccupants: make(map[grid.Point]int64),
	}, nil
}

// AddUnit creates a world unit for a player when absent.
func (world *World) AddUnit(playerID int64) {
	if _, exists := world.units[playerID]; exists {
		return
	}
	roomUnit, err := worldunit.New(worldunit.Params{
		ID: world.nextUnitID, OwnerID: playerID, Kind: worldunit.KindPlayer,
		Position: world.door, Body: world.body, Head: world.head,
	})
	if err != nil {
		return
	}
	world.units[playerID] = roomUnit
	world.nextUnitID++
}

// RemoveUnit removes a player world unit and releases its furniture slot.
func (world *World) RemoveUnit(playerID int64) {
	world.releaseSlot(playerID)
	delete(world.units, playerID)
}

// ClearUnits removes every world unit and slot reservation.
func (world *World) ClearUnits() {
	world.units = make(map[int64]*worldunit.Unit)
	world.unitSlots = make(map[int64]grid.Point)
	world.slotOccupants = make(map[grid.Point]int64)
}

// Door returns the room entry position.
func (world *World) Door() worldpath.Position {
	return world.door
}

// ReloadFixtures replaces fixtures for one source.
func (world *World) ReloadFixtures(sourceID int64, fixtures []surface.Fixture) error {
	return world.resolver.ReplaceFixtures(sourceID, fixtures)
}

// ReloadFurniture replaces one furniture snapshot and reconciles occupied slots.
func (world *World) ReloadFurniture(sourceID int64, item *worldfurniture.Item) ([]UnitSnapshot, error) {
	var fixtures []surface.Fixture
	if item != nil {
		built, err := worldfurniture.Fixtures(*item)
		if err != nil {
			return nil, err
		}
		fixtures = built
	}
	previous, hadPrevious := world.furniture[sourceID]
	if err := world.resolver.ReplaceFixtures(sourceID, fixtures); err != nil {
		return nil, err
	}
	if item != nil {
		world.furniture[sourceID] = *item
	} else {
		delete(world.furniture, sourceID)
	}
	if !hadPrevious {
		return nil, nil
	}

	return world.reconcileSlotOccupants(worldfurniture.Slots(previous), item), nil
}

// furnitureFixtures converts furniture into resolver fixtures and an id index.
func furnitureFixtures(items []worldfurniture.Item) ([]surface.Fixture, map[int64]worldfurniture.Item, error) {
	fixtures := make([]surface.Fixture, 0, len(items))
	index := make(map[int64]worldfurniture.Item, len(items))
	for _, item := range items {
		itemFixtures, err := worldfurniture.Fixtures(item)
		if err != nil {
			return nil, nil, fmt.Errorf("build fixtures for furniture item %d: %w", item.ID, err)
		}
		fixtures = append(fixtures, itemFixtures...)
		index[item.ID] = item
	}

	return fixtures, index, nil
}
