package main

import (
	"log"
	"net/http"

	authconfig "github.com/KasumiMercury/primind-central-backend/internal/auth/config"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/controller/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	authsvc "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/service"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

func main() {
	authCfg, err := authconfig.Load()
	if err != nil {
		log.Fatalf("failed to load auth config: %v", err)
	}

	mux := http.NewServeMux()

	paramsRepo := repository.NewInMemoryOIDCParamsRepository()
	paramsCtrl := oidc.NewParamsUseCase(authCfg.OIDC, paramsRepo)
	authService := authsvc.NewService(authCfg, paramsCtrl)

	authPath, authHandler := authv1connect.NewAuthServiceHandler(authService)
	mux.Handle(authPath, authHandler)

	addr := ":8080"
	log.Printf("starting Connect API server on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
