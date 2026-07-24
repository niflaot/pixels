package admin

import (
	"context"

	sdkcommand "github.com/niflaot/pixels/sdk/command"
	"go.minekube.com/brigodier"
)

// alertCommand builds the direct alert command tree.
func alertCommand(service *Service) brigodier.LiteralNodeBuilder {
	return brigodier.Literal("alert").
		Requires(sdkcommand.RequiresPermission(string(AlertPermission))).
		Then(brigodier.Argument("player", brigodier.StringWord).
			Then(brigodier.Argument("reason", brigodier.StringPhrase).
				Executes(commandExecution(func(ctx context.Context, sender sdkcommand.Sender, call *brigodier.CommandContext) error {
					return service.Alert(ctx, sender, call.String("player"), call.String("reason"))
				}))))
}

// hotelAlertCommand builds the hotel-wide alert command tree.
func hotelAlertCommand(service *Service) brigodier.LiteralNodeBuilder {
	return brigodier.Literal("halert").
		Requires(sdkcommand.RequiresPermission(string(HotelAlertPermission))).
		Then(brigodier.Argument("reason", brigodier.StringPhrase).
			Executes(commandExecution(func(ctx context.Context, sender sdkcommand.Sender, call *brigodier.CommandContext) error {
				return service.HotelAlert(ctx, sender, call.String("reason"))
			})))
}

// aboutCommand builds the build metadata command tree.
func aboutCommand(service *Service) brigodier.LiteralNodeBuilder {
	return brigodier.Literal("about").
		Requires(sdkcommand.RequiresPermission(string(AboutPermission))).
		Executes(commandExecution(func(ctx context.Context, sender sdkcommand.Sender, _ *brigodier.CommandContext) error {
			return service.About(ctx, sender)
		}))
}

// traceCommand builds the packet trace toggle command tree.
func traceCommand(service *Service) brigodier.LiteralNodeBuilder {
	return brigodier.Literal("trace").
		Requires(sdkcommand.RequiresPermission(string(TracePermission))).
		Executes(commandExecution(func(ctx context.Context, sender sdkcommand.Sender, _ *brigodier.CommandContext) error {
			return service.ToggleTrace(ctx, sender)
		}))
}

// commandExecution resolves one required sender before domain execution.
func commandExecution(execute func(context.Context, sdkcommand.Sender, *brigodier.CommandContext) error) brigodier.Command {
	return brigodier.CommandFunc(func(call *brigodier.CommandContext) error {
		sender, found := sdkcommand.SenderFrom(call.Context)
		if !found {
			return nil
		}

		return execute(call.Context, sender, call)
	})
}
