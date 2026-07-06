package navigator

import (
	"testing"

	navclosed "github.com/niflaot/pixels/internal/realm/navigator/events/closed"
	navfavorite "github.com/niflaot/pixels/internal/realm/navigator/events/favoritechanged"
	navinitialized "github.com/niflaot/pixels/internal/realm/navigator/events/initialized"
	navsearch "github.com/niflaot/pixels/internal/realm/navigator/events/searchexecuted"
	"github.com/niflaot/pixels/internal/realm/navigator/service"
)

// TestEventNames verifies navigator event names are stable.
func TestEventNames(t *testing.T) {
	events := []string{
		string(navinitialized.Name),
		string(navclosed.Name),
		string(navsearch.Name),
		string(navfavorite.Name),
	}

	for _, event := range events {
		if event == "" {
			t.Fatal("expected event name")
		}
	}
}

// TestProvidersExposeContracts verifies module helper providers return contracts.
func TestProvidersExposeContracts(t *testing.T) {
	navigatorService := service.New(nil)

	if NewStore(nil) == nil {
		t.Fatal("expected store")
	}
	if NewManager(navigatorService) == nil {
		t.Fatal("expected navigator manager")
	}
}
