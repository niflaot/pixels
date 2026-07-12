package navigator

import (
	realmconn "github.com/niflaot/pixels/internal/realm/connection"
	countscmd "github.com/niflaot/pixels/internal/realm/navigator/browse/category/counts"
	eventcatscmd "github.com/niflaot/pixels/internal/realm/navigator/browse/category/events"
	flatcatscmd "github.com/niflaot/pixels/internal/realm/navigator/browse/category/list"
	forwardcmd "github.com/niflaot/pixels/internal/realm/navigator/browse/room/forward"
	infocmd "github.com/niflaot/pixels/internal/realm/navigator/browse/room/info"
	navruntime "github.com/niflaot/pixels/internal/realm/navigator/browse/runtime"
	searchcmd "github.com/niflaot/pixels/internal/realm/navigator/browse/search"
	navservice "github.com/niflaot/pixels/internal/realm/navigator/core"
	cancreatecmd "github.com/niflaot/pixels/internal/realm/navigator/create/check"
	createcmd "github.com/niflaot/pixels/internal/realm/navigator/create/room"
	initcmd "github.com/niflaot/pixels/internal/realm/navigator/session/init"
	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	roommoderation "github.com/niflaot/pixels/internal/realm/room/control/moderation"
	roomrights "github.com/niflaot/pixels/internal/realm/room/control/rights"
	roomservice "github.com/niflaot/pixels/internal/realm/room/record/service"
	roomlive "github.com/niflaot/pixels/internal/realm/room/runtime/live"
	"github.com/niflaot/pixels/internal/realm/session/binding"
	"github.com/niflaot/pixels/pkg/bus"
	"github.com/niflaot/pixels/pkg/i18n"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RegisterConnectionHandlers registers navigator packet handlers.
func RegisterConnectionHandlers(handlers *realmconn.Handlers, deps HandlerDeps) {
	if handlers == nil || handlers.Inbound == nil {
		return
	}

	initcmd.RegisterPacketHandler(handlers.Inbound, initcmd.NewPacketHandler(initcmd.Handler{
		Players:   deps.Players,
		Bindings:  deps.Bindings,
		Navigator: deps.Navigator,
		Rooms:     deps.Rooms,
		Events:    deps.Events,
	}, deps.Log))
	searchcmd.RegisterPacketHandler(handlers.Inbound, searchcmd.NewPacketHandler(searchcmd.Handler{
		Players:   deps.Players,
		Bindings:  deps.Bindings,
		Navigator: deps.Navigator,
		Rooms:     deps.Rooms,
		Runtime:   deps.Runtime,
		Rights:    deps.Rights,
		Events:    deps.Events,
	}, deps.Log))
	cancreatecmd.RegisterPacketHandler(handlers.Inbound, cancreatecmd.NewPacketHandler(cancreatecmd.Handler{
		Players:  deps.Players,
		Bindings: deps.Bindings,
		Rooms:    deps.Rooms,
	}, deps.Log))
	createcmd.RegisterPacketHandler(handlers.Inbound, createcmd.NewPacketHandler(createcmd.Handler{
		Players:      deps.Players,
		Bindings:     deps.Bindings,
		Rooms:        deps.Rooms,
		Events:       deps.Events,
		Translations: deps.Translations,
		Log:          deps.Log,
	}, deps.Log))
	infocmd.RegisterPacketHandler(handlers.Inbound, infocmd.NewPacketHandler(infocmd.Handler{
		Players:    deps.Players,
		Bindings:   deps.Bindings,
		Rooms:      deps.Rooms,
		Runtime:    deps.Runtime,
		Moderation: deps.Moderation,
	}, deps.Log))
	forwardcmd.RegisterPacketHandler(handlers.Inbound, forwardcmd.NewPacketHandler(forwardcmd.Handler{
		Players:  deps.Players,
		Bindings: deps.Bindings,
		Rooms:    deps.Rooms,
	}, deps.Log))
	flatcatscmd.RegisterPacketHandler(handlers.Inbound, flatcatscmd.NewPacketHandler(flatcatscmd.Handler{
		Players:    deps.Players,
		Bindings:   deps.Bindings,
		Categories: deps.Rooms,
	}, deps.Log))
	eventcatscmd.RegisterPacketHandler(handlers.Inbound, eventcatscmd.NewPacketHandler(eventcatscmd.Handler{
		Players:  deps.Players,
		Bindings: deps.Bindings,
	}, deps.Log))
	countscmd.RegisterPacketHandler(handlers.Inbound, countscmd.NewPacketHandler(countscmd.Handler{
		Players:  deps.Players,
		Bindings: deps.Bindings,
		Counts:   deps.Counts,
	}, deps.Log))
}

// HandlerDeps contains navigator handler dependencies.
type HandlerDeps struct {
	fx.In

	// Players stores live player state.
	Players *playerlive.Registry
	// Bindings stores player connection bindings.
	Bindings *binding.Registry
	// Navigator manages navigator persistence.
	Navigator navservice.Manager
	// Rooms manages room persistence.
	Rooms roomservice.Manager
	// Runtime stores active room runtime state.
	Runtime *roomlive.Registry
	// Rights resolves persistent room-scoped visibility.
	Rights roomrights.Manager
	// Moderation resolves viewer room moderation capability.
	Moderation roommoderation.Manager
	// Counts stores current navigator category counts.
	Counts *navruntime.CategoryCountBroadcaster
	// Events publishes realm events.
	Events *bus.Bus
	// Log records command dispatch.
	Log *zap.Logger
	// Translations resolves localized navigator feedback.
	Translations i18n.Translator
}
