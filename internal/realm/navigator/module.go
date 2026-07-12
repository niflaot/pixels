// Package navigator contains navigator realm persistence and runtime wiring.
package navigator

import (
	navruntime "github.com/niflaot/pixels/internal/realm/navigator/browse/runtime"
	"github.com/niflaot/pixels/internal/realm/navigator/core"
	"github.com/niflaot/pixels/internal/realm/navigator/database"
	"github.com/niflaot/pixels/internal/realm/navigator/record"
	"github.com/niflaot/pixels/pkg/postgres"
	"go.uber.org/fx"
)

// Module provides navigator realm persistence state.
var Module = fx.Module(
	"realm-navigator",
	fx.Provide(
		NewStore,
		navruntime.NewCategoryCountBroadcaster,
		navruntime.NewRoomCountBroadcaster,
		core.New,
		NewManager,
	),
	fx.Invoke(RegisterConnectionHandlers),
	fx.Invoke(navruntime.RegisterCategoryCounts),
	fx.Invoke(navruntime.RegisterRoomCounts),
)

// NewStore creates the navigator persistence store.
func NewStore(pool *postgres.Pool) record.Store {
	return database.New(pool)
}

// NewManager exposes the navigator management contract.
func NewManager(service *core.Service) core.Manager {
	return service
}
