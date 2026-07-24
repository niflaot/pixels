package admin

import (
	plugincommand "github.com/niflaot/pixels/internal/plugin/command"
	pluginruntime "github.com/niflaot/pixels/internal/plugin/runtime"
	admintrace "github.com/niflaot/pixels/internal/realm/admin/trace"
	"go.uber.org/fx"
)

// coreScope owns built-in command roots for the process lifetime.
var coreScope = pluginruntime.NewScope("core")

// Module provides first-party administrative chat commands.
var Module = fx.Module(
	"realm-admin",
	fx.Provide(admintrace.New, New),
	fx.Invoke(admintrace.Register, RegisterCommands),
)

// RegisterCommands claims and registers every first-party command root.
func RegisterCommands(tree *plugincommand.Tree, service *Service) error {
	access := plugincommand.NewAccess(tree, coreScope)
	if err := access.Register(alertCommand(service)); err != nil {
		return err
	}
	if err := access.Register(hotelAlertCommand(service)); err != nil {
		return err
	}
	if err := access.Register(aboutCommand(service)); err != nil {
		return err
	}

	return access.Register(traceCommand(service))
}
