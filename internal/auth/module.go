package auth

import (
	"context"
	"fmt"
	"log/slog"
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
	logger := slog.Default().WithGroup("auth")

	logger.Debug("loading auth configuration")

	authCfg, err := authconfig.Load()
	if err != nil {
		logger.Error("failed to load auth config", slog.String("error", err.Error()))

		return "", nil, err
	}

	paramsRepo := repository.NewInMemoryOIDCParamsRepository()
	sessionRepo := repository.NewInMemorySessionRepository()

	var (
		paramsGenerator appoidc.OIDCParamsGenerator
		providers       map[domainoidc.ProviderID]*infraoidc.RPProvider
	)

	if authCfg.OIDC != nil {
		logger.Debug("initializing oidc providers")

		providers = make(map[domainoidc.ProviderID]*infraoidc.RPProvider)

		for providerID, providerCfg := range authCfg.OIDC.Providers {
			rpProvider, err := infraoidc.NewRPProvider(ctx, providerCfg)
			if err != nil {
				logger.Error(
					"failed to initialize oidc provider",
					slog.String("provider", string(providerID)),
					slog.String("error", err.Error()),
				)

				return "", nil, fmt.Errorf("failed to initialize OIDC provider %s: %w", providerID, err)
			}

			providers[providerID] = rpProvider
			logger.Info("initialized oidc provider", slog.String("provider", string(providerID)))
		}

		appProviders := make(map[domainoidc.ProviderID]appoidc.OIDCProvider)
		for id, p := range providers {
			appProviders[id] = p
		}

		paramsGenerator = appoidc.NewParamsGenerator(appProviders, paramsRepo)
	} else {
		logger.Warn("oidc configuration not provided; auth endpoints will be disabled")
	}

	var loginHandler appoidc.OIDCLoginUseCase

	if authCfg.Session != nil && authCfg.OIDC != nil {
		jwtGenerator := sessionjwt.NewSessionJWTGenerator(authCfg.Session)

		appProviders := make(map[domainoidc.ProviderID]appoidc.OIDCProviderWithLogin)
		for id, p := range providers {
			appProviders[id] = p
		}

		loginHandler = appoidc.NewLoginHandler(appProviders, paramsRepo, sessionRepo, jwtGenerator, authCfg.Session)

		logger.Info("login handler initialized")
	} else {
		logger.Warn("session or oidc config missing; login handler disabled")
	}

	authService := authsvc.NewService(paramsGenerator, loginHandler)

	authPath, authHandler := authv1connect.NewAuthServiceHandler(authService)
	logger.Info("auth service handler registered", slog.String("path", authPath))

	return authPath, authHandler, nil
}
