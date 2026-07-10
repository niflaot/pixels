package leave

import (
	"context"
	"testing"

	roomlive "github.com/niflaot/pixels/internal/realm/room/live"
	netconn "github.com/niflaot/pixels/networking/connection"
	outdesktop "github.com/niflaot/pixels/networking/outbound/session/desktop"
)

// TestToDesktopUsesStandardLeaveAndSendsHotelView verifies complete door-exit teardown.
func TestToDesktopUsesStandardLeaveAndSendsHotelView(t *testing.T) {
	player := playerForTest(t)
	players := playerRegistryForTest(t, player)
	connections := netconn.NewRegistry()
	sent := registerConnectionForTest(t, connections, "conn")
	runtime := roomlive.NewRegistry(nil)
	active := activeRoomForTest(t, runtime)

	err := (Handler{Players: players, Runtime: runtime, Connections: connections}).ToDesktop(context.Background(), 7)
	if err != nil {
		t.Fatalf("leave to desktop: %v", err)
	}
	if active.Occupancy().Count != 0 {
		t.Fatalf("expected empty room, got %#v", active.Occupancy())
	}
	if len(*sent) != 1 || (*sent)[0].Header != outdesktop.Header {
		t.Fatalf("expected desktop packet, got %#v", *sent)
	}
}
