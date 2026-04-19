package auth

import (
	"go.uber.org/fx"

	"github.com/vitaliizinchenko/lab/internal/auth/handlers"
	"github.com/vitaliizinchenko/lab/internal/auth/repos"
)

// Module provides all auth-domain constructors to the fx application.
var Module = fx.Module("auth",
	fx.Provide(repos.NewUserRepository),
	fx.Provide(repos.NewApiKeyRepository),
	fx.Provide(handlers.NewTokenHandler),
	fx.Provide(handlers.NewApiKeysHandler),
	fx.Provide(handlers.NewUsersHandler),
	fx.Provide(handlers.NewLoginHandler),
)
