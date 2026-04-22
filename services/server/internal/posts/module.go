package posts

import (
	"go.uber.org/fx"

	"github.com/vitaliizinchenko/lab/internal/posts/handlers"
	"github.com/vitaliizinchenko/lab/internal/posts/repos"
)

// Module provides all posts-domain constructors to the fx application.
var Module = fx.Module("posts",
	fx.Provide(repos.NewPostRepository),
	fx.Provide(handlers.NewPostsHandler),
)
