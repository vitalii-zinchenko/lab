package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/vitaliizinchenko/lab/gen"
	"github.com/vitaliizinchenko/lab/handler"
)

func main() {
	swagger, err := gen.GetSwagger()
	if err != nil {
		log.Fatalf("failed to load swagger spec: %v", err)
	}
	// Clear servers so the validator doesn't enforce host/scheme matching
	swagger.Servers = nil

	router := gin.Default()

	// Validate all incoming requests against the OpenAPI spec
	router.Use(ginmiddleware.OapiRequestValidator(swagger))

	h := handler.New()
	strictHandler := gen.NewStrictHandler(h, nil)
	gen.RegisterHandlers(router, strictHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("Server listening on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
