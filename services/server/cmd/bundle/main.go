package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile("cmd/server/spec.yaml")
	if err != nil {
		log.Fatalf("load spec: %v", err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		log.Fatalf("validate spec: %v", err)
	}

	doc.InternalizeRefs(context.Background(), nil)

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}

	if err := os.WriteFile("cmd/server/openapi.gen.json", out, 0644); err != nil {
		log.Fatalf("write: %v", err)
	}
}
