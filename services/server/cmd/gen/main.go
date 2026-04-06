package main

import (
	"gorm.io/gen"

	"github.com/vitaliizinchenko/lab/model"
)

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath:      "./repository/query",
		ModelPkgPath: "./model",
		Mode:         gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
	})

	g.ApplyBasic(
		model.Item{},
		model.EventHistory{},
		model.User{},
		model.ApiKey{},
	)

	g.Execute()
}
