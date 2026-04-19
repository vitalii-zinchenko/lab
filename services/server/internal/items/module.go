package items

import (
	"go.uber.org/fx"

	"github.com/vitaliizinchenko/lab/internal/items/handlers"
	"github.com/vitaliizinchenko/lab/internal/items/repos"
)

// Module provides all items-domain constructors to the fx application.
var Module = fx.Module("items",
	fx.Provide(repos.NewItemRepository),
	fx.Provide(handlers.NewItemsHandler),
)
