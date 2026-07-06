package room

import (
	"testing"

	roomcreated "github.com/niflaot/pixels/internal/realm/room/events/created"
	roomdeleted "github.com/niflaot/pixels/internal/realm/room/events/deleted"
	roomentered "github.com/niflaot/pixels/internal/realm/room/events/entered"
	roomleft "github.com/niflaot/pixels/internal/realm/room/events/left"
	roomoccupancy "github.com/niflaot/pixels/internal/realm/room/events/occupancychanged"
	roomupdated "github.com/niflaot/pixels/internal/realm/room/events/updated"
	"github.com/niflaot/pixels/internal/realm/room/layout"
	"github.com/niflaot/pixels/internal/realm/room/service"
	"github.com/niflaot/pixels/pkg/bus"
)

// TestEventNames verifies room event names are stable.
func TestEventNames(t *testing.T) {
	events := []string{
		string(roomcreated.Name),
		string(roomupdated.Name),
		string(roomdeleted.Name),
		string(roomoccupancy.Name),
		string(roomentered.Name),
		string(roomleft.Name),
	}

	for _, event := range events {
		if event == "" {
			t.Fatal("expected event name")
		}
	}
}

// TestProvidersExposeContracts verifies module helper providers return contracts.
func TestProvidersExposeContracts(t *testing.T) {
	layoutService := layout.NewService(nil)
	roomService := service.New(nil, layoutService)

	if NewLiveRegistry(bus.New()) == nil {
		t.Fatal("expected live registry")
	}
	if NewLayoutStore(nil) == nil {
		t.Fatal("expected layout store")
	}
	if NewStore(nil) == nil {
		t.Fatal("expected room store")
	}
	if NewLayoutManager(layoutService) == nil {
		t.Fatal("expected layout manager")
	}
	if NewManager(roomService) == nil {
		t.Fatal("expected room manager")
	}
}
