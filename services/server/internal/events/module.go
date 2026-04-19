package events

import (
	"go.uber.org/fx"

	"github.com/vitaliizinchenko/lab/internal/events/handlers"
	"github.com/vitaliizinchenko/lab/internal/events/repos"
)

// Module provides all events-domain constructors to the fx application.
var Module = fx.Module("events",
	fx.Provide(repos.NewEventHistoryRepository),
	fx.Provide(repos.NewChEventRepository),
	fx.Provide(handlers.NewEventsHandler),
	fx.Provide(handlers.NewChEventsHandler),
)
