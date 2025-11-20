package main

import (
	"log"
	"net/http"

	authsvc "github.com/KasumiMercury/primind-central-backend/internal/auth"
	authconfig "github.com/KasumiMercury/primind-central-backend/internal/auth/config"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

func main() {
	authCfg, err := authconfig.Load()
	if err != nil {
		log.Fatalf("failed to load auth config: %v", err)
	}

	mux := http.NewServeMux()

	authService := authsvc.NewService(authCfg)

	authPath, authHandler := authv1connect.NewAuthServiceHandler(authService)
	mux.Handle(authPath, authHandler)

	addr := ":8080"
	log.Printf("starting Connect API server on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
