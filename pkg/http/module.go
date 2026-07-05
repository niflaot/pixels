package http

import "go.uber.org/fx"

// Module provides Fiber HTTP transport to an Fx dependency graph.
var Module = fx.Module(
	"http",
	fx.Provide(New),
	fx.Invoke(Start),
)
