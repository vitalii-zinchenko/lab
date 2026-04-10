package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// refNameResolver extracts the bare schema name from a $ref.
// For example, "../items/spec_schemas.yaml#/components/schemas/Item" → "Item".
func refNameResolver(_ *openapi3.T, ref openapi3.ComponentRef) string {
	if u := ref.RefPath(); u != nil && u.Fragment != "" {
		parts := strings.Split(u.Fragment, "/")
		return parts[len(parts)-1]
	}
	s := ref.RefString()
	if idx := strings.LastIndexAny(s, "/#"); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

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

	doc.InternalizeRefs(context.Background(), refNameResolver)

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}

	if err := os.WriteFile("cmd/server/openapi.gen.json", out, 0644); err != nil {
		log.Fatalf("write: %v", err)
	}
}
