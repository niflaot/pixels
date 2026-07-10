// Package room contains room realm persistence and runtime wiring.
package room

import (
	permissionservice "github.com/niflaot/pixels/internal/permission/service"
	roomentry "github.com/niflaot/pixels/internal/realm/room/entry"
	"github.com/niflaot/pixels/internal/realm/room/layout"
	"github.com/niflaot/pixels/internal/realm/room/repository"
	"github.com/niflaot/pixels/internal/realm/room/service"
	"github.com/niflaot/pixels/pkg/i18n"
	"github.com/niflaot/pixels/pkg/postgres"
	"github.com/niflaot/pixels/pkg/redis"
	"go.uber.org/fx"
)

// Module provides room realm persistence state.
var Module = fx.Module(
	"realm-room",
	fx.Provide(
		NewLayoutStore,
		NewStore,
		layout.NewService,
		service.New,
		NewLiveRegistry,
		NewLayoutManager,
		NewManager,
		NewEntryService,
	),
	fx.Invoke(RegisterRuntimeCleanup),
	fx.Invoke(RegisterConnectionHandlers),
)

// NewEntryService creates closed-room entry behavior.
func NewEntryService(config roomentry.Config, redisClient *redis.Client, permissions permissionservice.Checker, translations i18n.Translator) *roomentry.Service {
	return roomentry.New(config, redisClient, permissions, translations, roomentry.Nodes{
		EnterAny: EnterAny, EnterFull: EnterFull, AnswerAnyDoorbell: AnswerAnyDoorbell,
	})
}

// NewLayoutStore creates the room layout persistence store.
func NewLayoutStore(pool *postgres.Pool) layout.Store {
	return layout.NewRepository(pool)
}

// NewStore creates the room persistence store.
func NewStore(pool *postgres.Pool) repository.Store {
	return repository.New(pool)
}

// NewLayoutManager exposes the room layout management contract.
func NewLayoutManager(service *layout.Service) layout.Manager {
	return service
}

// NewManager exposes the room management contract.
func NewManager(roomService *service.Service) service.Manager {
	return roomService
}
