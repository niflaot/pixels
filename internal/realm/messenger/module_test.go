package messenger

import (
	"testing"

	realmconn "github.com/niflaot/pixels/internal/realm/connection"
	netconn "github.com/niflaot/pixels/networking/connection"
)

// TestRegisterConnectionHandlersRegistersEveryNativePacket verifies messenger wiring.
func TestRegisterConnectionHandlersRegistersEveryNativePacket(t *testing.T) {
	registry := netconn.NewHandlerRegistry()
	RegisterConnectionHandlers(&realmconn.Handlers{Inbound: registry}, HandlerDeps{})
	if registry.Len() != 21 {
		t.Fatalf("expected twenty-one native messenger handlers, got %d", registry.Len())
	}
}

// TestRegisterConnectionHandlersAllowsMissingRegistry verifies optional wiring safety.
func TestRegisterConnectionHandlersAllowsMissingRegistry(t *testing.T) {
	RegisterConnectionHandlers(nil, HandlerDeps{})
}
