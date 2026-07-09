package routes

import (
	"github.com/gofiber/fiber/v2"
	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	netconn "github.com/niflaot/pixels/networking/connection"
	"github.com/niflaot/pixels/pkg/i18n"
)

const basePath = "/api/admin/players"

// Register mounts protected player notification administration routes.
func Register(app *fiber.App, players *playerlive.Registry, connections *netconn.Registry, translations i18n.Translator) {
	app.Post(basePath+"/:id/notifications", notifyHandler(players, connections, translations))
}
