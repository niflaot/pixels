package connection

import (
	"github.com/niflaot/pixels/networking/crypto/diffie"
	"go.uber.org/fx"
)

// Module provides connection-realm handlers.
var Module = fx.Module(
	"realm-connection",
	fx.Provide(diffie.NewFactory, NewHandlersWithPermissions),
	fx.Invoke(
		RegisterSecurityTranslations,
		RegisterEffectHandlers,
		RegisterAchievementHandlers,
		RegisterProtocolCompatibilityHandlers,
		RegisterPlayerCompatibilityHandlers,
		RegisterPlayerSettingsHandlers,
		RegisterPlayerProfileHandlers,
		RegisterPlayerWardrobeHandlers,
		RegisterPlayerIdentityHandlers,
	),
)
