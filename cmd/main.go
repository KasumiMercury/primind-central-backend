package main

import (
	"context"
	"log"
	"net/http"

	authconfig "github.com/KasumiMercury/primind-central-backend/internal/auth/config"
	oidcctrl "github.com/KasumiMercury/primind-central-backend/internal/auth/controller/oidc"
	infraoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	authsvc "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/service"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

func main() {
	ctx := context.Background()

	authCfg, err := authconfig.Load()
	if err != nil {
		log.Fatalf("failed to load auth config: %v", err)
	}

	mux := http.NewServeMux()

	paramsRepo := repository.NewInMemoryOIDCParamsRepository()
	var paramsGenerator oidcctrl.OIDCParamsGenerator
	if authCfg.OIDC != nil {
		paramsGenerator, err = infraoidc.NewParamsGenerator(ctx, authCfg.OIDC, paramsRepo)
		if err != nil {
			log.Fatalf("failed to initialize OIDC params generator: %v", err)
		}
	}

	authService := authsvc.NewService(authCfg, paramsGenerator)

	authPath, authHandler := authv1connect.NewAuthServiceHandler(authService)
	mux.Handle(authPath, authHandler)

	addr := ":8080"
	log.Printf("starting Connect API server on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
