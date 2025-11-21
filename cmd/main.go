package main

import (
	"context"
	"log"
	"net/http"

	authmodule "github.com/KasumiMercury/primind-central-backend/internal/auth"
)

func main() {
	ctx := context.Background()
	mux := http.NewServeMux()

	authPath, authHandler, err := authmodule.NewHTTPHandler(ctx)
	if err != nil {
		log.Fatalf("failed to initialize auth service: %v", err)
	}
	mux.Handle(authPath, authHandler)

	addr := ":8080"
	log.Printf("starting Connect API server on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
