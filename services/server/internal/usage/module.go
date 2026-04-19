package usage

import (
	"go.uber.org/fx"

	"github.com/vitaliizinchenko/lab/internal/usage/handlers"
	"github.com/vitaliizinchenko/lab/internal/usage/repos"
)

// Module provides all usage-domain constructors to the fx application.
var Module = fx.Module("usage",
	fx.Provide(repos.NewPostgresUsageRepository),
	fx.Provide(handlers.NewUsageHandler),
)
