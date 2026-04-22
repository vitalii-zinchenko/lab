package server

import (
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.uber.org/fx"

	authdomain "github.com/vitaliizinchenko/lab/internal/auth"
	eventsdomain "github.com/vitaliizinchenko/lab/internal/events"
	itemsdomain "github.com/vitaliizinchenko/lab/internal/items"
	postsdomain "github.com/vitaliizinchenko/lab/internal/posts"
	usagedomain "github.com/vitaliizinchenko/lab/internal/usage"
)

// newPrometheusMiddleware builds the Gin Prometheus middleware.
// ReqCntURLLabelMappingFn uses the route template (e.g. /items/:id) instead
// of the actual path, preventing cardinality explosion from UUID path params.
func newPrometheusMiddleware() gin.HandlerFunc {
	p := ginprometheus.NewPrometheus("gin")
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		return c.FullPath()
	}
	return p.HandlerFunc()
}

// AppModule is the composable fx option set that wires all domain modules,
// the router, and cross-cutting providers. It can be embedded in both the
// production main and integration tests.
//
// Callers are responsible for providing the infrastructure dependencies that
// AppModule does not supply:
//
//	*gorm.DB  — PostgreSQL connection
//	*sql.DB   — ClickHouse connection
//	[]byte    — JWT secret
var AppModule = fx.Options(
	fx.Provide(
		fx.Annotate(newPrometheusMiddleware, fx.ResultTags(`name:"prometheus"`)),
	),
	authdomain.Module,
	eventsdomain.Module,
	itemsdomain.Module,
	postsdomain.Module,
	usagedomain.Module,
	Module, // server.Module — provides NewRouter
)
