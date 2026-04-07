package main

import (
	"gorm.io/gen"

	authmodels "github.com/vitaliizinchenko/lab/internal/auth/models"
	eventmodels "github.com/vitaliizinchenko/lab/internal/events/models"
	itemmodels "github.com/vitaliizinchenko/lab/internal/items/models"
)

func main() {
	// Items
	g := gen.NewGenerator(gen.Config{
		OutPath:      "./internal/items/repos/query",
		ModelPkgPath: "./internal/items/models",
		Mode:         gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
	})
	g.ApplyBasic(itemmodels.Item{})
	g.Execute()

	// Events
	g = gen.NewGenerator(gen.Config{
		OutPath:      "./internal/events/repos/query",
		ModelPkgPath: "./internal/events/models",
		Mode:         gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
	})
	g.ApplyBasic(eventmodels.EventHistory{})
	g.Execute()

	// Auth
	g = gen.NewGenerator(gen.Config{
		OutPath:      "./internal/auth/repos/query",
		ModelPkgPath: "./internal/auth/models",
		Mode:         gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
	})
	g.ApplyBasic(authmodels.User{}, authmodels.ApiKey{})
	g.Execute()
}
