package server

import "go.uber.org/fx"

// Module provides the HTTP router constructor to the fx application.
var Module = fx.Module("server",
	fx.Provide(NewRouter),
)
