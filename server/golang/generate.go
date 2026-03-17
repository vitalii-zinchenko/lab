//go:build ignore

package main

//go:generate oapi-codegen --config=oapi-codegen.yaml api/openapi.yaml
//go:generate go run ./cmd/gen
