// Package connection contains connection-realm handlers and commands.
package connection

import (
	"github.com/niflaot/pixels/internal/auth/sso"
	"github.com/niflaot/pixels/internal/realm/connection/handlers/handshake"
	"github.com/niflaot/pixels/internal/realm/connection/handlers/heartbeat"
	"github.com/niflaot/pixels/internal/realm/connection/handlers/latency"
	"github.com/niflaot/pixels/internal/realm/connection/handlers/security"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
)

// Handlers contains connection-realm handler registries.
type Handlers struct {
	// Inbound routes client packets.
	Inbound *netconn.HandlerRegistry
	// Outbound routes server packets.
	Outbound *netconn.HandlerRegistry
}

// NewHandlers creates connection-realm handler registries.
func NewHandlers(sso *sso.Service) *Handlers {
	inbound := netconn.NewHandlerRegistry()
	outbound := netconn.NewHandlerRegistry()
	handlers := &Handlers{Inbound: inbound, Outbound: outbound}

	registerInbound(inbound, sso)
	outbound.SetFallback(noopHandler, netconn.AllowAnyActiveState(), netconn.AllowUnauthenticated())

	return handlers
}

// registerInbound registers connection-realm inbound handlers.
func registerInbound(registry *netconn.HandlerRegistry, service *sso.Service) {
	handshake.Register(registry)
	security.Register(registry, service)
	heartbeat.Register(registry)
	latency.Register(registry)
}

// noopHandler accepts outbound packets without side effects.
func noopHandler(netconn.Context, codec.Packet) error {
	return nil
}
