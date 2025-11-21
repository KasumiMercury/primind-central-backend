package auth

import (
	"context"
	"fmt"
	"net/http"

	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	authconfig "github.com/KasumiMercury/primind-central-backend/internal/auth/config"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	sessionjwt "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/jwt"
	infraoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	authsvc "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/service"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

// NewHTTPHandler wires the auth module and returns the Connect HTTP handler
// and its base path for registration into an HTTP mux.
func NewHTTPHandler(ctx context.Context) (string, http.Handler, error) {
	authCfg, err := authconfig.Load()
	if err != nil {
		return "", nil, err
	}

	paramsRepo := repository.NewInMemoryOIDCParamsRepository()
	sessionRepo := repository.NewInMemorySessionRepository()

	var paramsGenerator appoidc.OIDCParamsGenerator
	var providers map[domainoidc.ProviderID]*infraoidc.RPProvider
	if authCfg.OIDC != nil {
		providers = make(map[domainoidc.ProviderID]*infraoidc.RPProvider)
		for providerID, providerCfg := range authCfg.OIDC.Providers {
			rpProvider, err := infraoidc.NewRPProvider(ctx, providerCfg)
			if err != nil {
				return "", nil, fmt.Errorf("failed to initialize OIDC provider %s: %w", providerID, err)
			}
			providers[providerID] = rpProvider
		}

		appProviders := make(map[domainoidc.ProviderID]appoidc.OIDCProvider)
		for id, p := range providers {
			appProviders[id] = p
		}

		paramsGenerator = appoidc.NewParamsGenerator(appProviders, paramsRepo)
	}

	var loginHandler appoidc.OIDCLoginUseCase
	if authCfg.Session != nil && authCfg.OIDC != nil {
		jwtGenerator := sessionjwt.NewSessionJWTGenerator(authCfg.Session)
		appProviders := make(map[domainoidc.ProviderID]appoidc.OIDCProviderWithLogin)
		for id, p := range providers {
			appProviders[id] = p
		}

		loginHandler = appoidc.NewLoginHandler(appProviders, paramsRepo, sessionRepo, jwtGenerator, authCfg.Session)
	}

	authService := authsvc.NewService(paramsGenerator, loginHandler)

	authPath, authHandler := authv1connect.NewAuthServiceHandler(authService)
	return authPath, authHandler, nil
}
