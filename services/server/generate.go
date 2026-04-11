//go:build ignore

package main

//go:generate go run ./cmd/bundle
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.yaml cmd/server/openapi.gen.json
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.client.yaml cmd/server/openapi.gen.json
//go:generate go run ./cmd/gen
//go:generate sh -c "cd cmd/server && npx --yes @apideck/portman --cliOptionsFile portman-cli-options.json"
