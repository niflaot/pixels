package messenger

import (
	permissionservice "github.com/niflaot/pixels/internal/permission/service"
	chatfilter "github.com/niflaot/pixels/internal/realm/chat/filter"
	realmconn "github.com/niflaot/pixels/internal/realm/connection"
	"github.com/niflaot/pixels/internal/realm/messenger/delivery"
	"github.com/niflaot/pixels/internal/realm/messenger/friends"
	"github.com/niflaot/pixels/internal/realm/messenger/presence"
	"github.com/niflaot/pixels/internal/realm/messenger/privacy"
	"github.com/niflaot/pixels/internal/realm/messenger/privatechat"
	"github.com/niflaot/pixels/internal/realm/messenger/profile"
	"github.com/niflaot/pixels/internal/realm/messenger/repository"
	"github.com/niflaot/pixels/internal/realm/messenger/service"
	"github.com/niflaot/pixels/internal/realm/messenger/session"
	"github.com/niflaot/pixels/internal/realm/messenger/social"
	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	playerservice "github.com/niflaot/pixels/internal/realm/player/service"
	roomlive "github.com/niflaot/pixels/internal/realm/room/runtime/live"
	"github.com/niflaot/pixels/pkg/bus"
	"github.com/niflaot/pixels/pkg/i18n"
	"github.com/niflaot/pixels/pkg/postgres"
	"github.com/niflaot/pixels/pkg/redis"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides messenger persistence, behavior, and packet routing.
var Module = fx.Module(
	"realm-messenger",
	fx.Provide(NewStore, NewPrivateChatWriter, NewService, delivery.New, presence.New, NewProfileBroadcaster),
	fx.Invoke(RegisterConnectionHandlers, presence.Register, profile.Register, privatechat.RegisterLifecycle),
)

// NewProfileBroadcaster creates targeted live profile relationship projection.
func NewProfileBroadcaster(messenger *service.Service, sender *delivery.Sender, log *zap.Logger) *profile.Broadcaster {
	return profile.New(messenger, sender, log)
}

// NewStore creates messenger persistence behavior.
func NewStore(pool *postgres.Pool) repository.Store {
	return repository.New(pool)
}

// NewService creates configured messenger behavior.
func NewService(config Config, store repository.Store, players playerservice.Manager, livePlayers *playerlive.Registry, rooms *roomlive.Registry, permissions permissionservice.Checker, redisClient *redis.Client, filter *chatfilter.Service, messageLog *privatechat.Writer) *service.Service {
	config = config.Normalize()
	return service.New(service.Options{
		MaxFriends: config.MaxFriends, MaxFriendsClub: config.MaxFriendsClub,
		MaxSearchResults: config.MaxSearchResults, SearchCacheTTL: config.SearchCacheTTL,
		SearchThrottle: config.SearchThrottle, ChatThrottle: config.ChatThrottle,
		ChatFilterEnabled: config.ChatFilterEnabled, ChatLogEnabled: config.ChatLogEnabled,
	}, store, players, livePlayers, rooms, permissions, redisClient, filter, service.Nodes{FriendsUnlimited: FriendsUnlimited, FollowAny: FollowAny}, messageLog)
}

// NewPrivateChatWriter creates optional asynchronous private-message persistence.
func NewPrivateChatWriter(config Config, store repository.Store, log *zap.Logger) *privatechat.Writer {
	return privatechat.New(privatechat.Config{Enabled: config.ChatLogEnabled}, store, log)
}

// HandlerDeps contains messenger packet handler dependencies.
type HandlerDeps struct {
	fx.In

	// Messenger stores messenger behavior.
	Messenger *service.Service
	// Delivery sends packets through authenticated bindings.
	Delivery *delivery.Sender
	// Events publishes completed messenger actions.
	Events bus.Publisher
	// Translations localizes hotel-facing feedback.
	Translations i18n.Translator
	// Log records command dispatch and unexpected failures.
	Log *zap.Logger
}

// RegisterConnectionHandlers registers every Nitro messenger inbound packet.
func RegisterConnectionHandlers(handlers *realmconn.Handlers, dependencies HandlerDeps) {
	if handlers == nil || handlers.Inbound == nil {
		return
	}
	session.RegisterHandlers(handlers.Inbound, session.Handler{Messenger: dependencies.Messenger, Delivery: dependencies.Delivery}, dependencies.Log)
	friends.RegisterHandlers(handlers.Inbound, friends.Handler{Messenger: dependencies.Messenger, Delivery: dependencies.Delivery, Events: dependencies.Events}, dependencies.Log)
	social.RegisterHandlers(handlers.Inbound, social.Handler{Messenger: dependencies.Messenger, Delivery: dependencies.Delivery, Events: dependencies.Events, Translations: dependencies.Translations}, dependencies.Log)
	privacy.RegisterHandlers(handlers.Inbound, privacy.Handler{Messenger: dependencies.Messenger, Delivery: dependencies.Delivery}, dependencies.Log)
}
