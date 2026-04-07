//go:build ignore

package main

//go:generate go run ./cmd/bundle
//go:generate oapi-codegen --config=oapi-codegen.yaml cmd/server/openapi.gen.json
//go:generate go run ./cmd/gen
