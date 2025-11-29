package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	applogout "github.com/KasumiMercury/primind-central-backend/internal/auth/app/logout"
	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	appsession "github.com/KasumiMercury/primind-central-backend/internal/auth/app/session"
	authconfig "github.com/KasumiMercury/primind-central-backend/internal/auth/config"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity"
	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	sessionjwt "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/jwt"
	infraoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/oidc"
	authsvc "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/service"
	authv1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1/authv1connect"
)

type Repositories struct {
	Params       domainoidc.ParamsRepository
	Sessions     domainsession.SessionRepository
	Users        user.UserRepository
	OIDCIdentity oidcidentity.OIDCIdentityRepository
	UserIdentity appoidc.UserWithOIDCIdentityRepository
}

// NewHTTPHandler wires the auth module and returns the Connect HTTP handler
// and its base path for registration into an HTTP mux.
func NewHTTPHandler(ctx context.Context, repos Repositories) (string, http.Handler, error) {
	logger := slog.Default().WithGroup("auth")

	logger.Debug("loading auth configuration")

	authCfg, err := authconfig.Load()
	if err != nil {
		logger.Error("failed to load auth config", slog.String("error", err.Error()))

		return "", nil, err
	}

	if repos.Params == nil || repos.Sessions == nil || repos.Users == nil || repos.OIDCIdentity == nil || repos.UserIdentity == nil {
		return "", nil, fmt.Errorf("repositories are not fully configured")
	}

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

		paramsGenerator = appoidc.NewParamsGenerator(appProviders, repos.Params)
	} else {
		logger.Warn("oidc configuration not provided; auth endpoints will be disabled")
	}

	var (
		loginHandler        appoidc.OIDCLoginUseCase
		jwtGenerator        *sessionjwt.SessionJWTGenerator
		sessionValidateCase appsession.ValidateSessionUseCase
		logoutHandler       applogout.LogoutUseCase
	)

	if authCfg.Session != nil && authCfg.OIDC != nil {
		jwtGenerator = sessionjwt.NewSessionJWTGenerator(authCfg.Session)
		jwtValidator := sessionjwt.NewSessionJWTValidator(authCfg.Session)

		appProviders := make(map[domainoidc.ProviderID]appoidc.OIDCProviderWithLogin)
		for id, p := range providers {
			appProviders[id] = p
		}

		loginHandler = appoidc.NewLoginHandler(
			appProviders,
			repos.Params,
			repos.Sessions,
			repos.Users,
			repos.OIDCIdentity,
			repos.UserIdentity,
			jwtGenerator,
			authCfg.Session,
		)
		sessionValidateCase = appsession.NewValidateSessionHandler(repos.Sessions, jwtValidator)
		logoutHandler = applogout.NewLogoutHandler(repos.Sessions, jwtValidator)

		logger.Info("login and session validation handlers initialized")
	} else {
		logger.Warn("session or oidc config missing; login and session validation handlers disabled")
	}

	authService := authsvc.NewService(paramsGenerator, loginHandler, sessionValidateCase, logoutHandler)

	authPath, authHandler := authv1connect.NewAuthServiceHandler(authService)
	logger.Info("auth service handler registered", slog.String("path", authPath))

	return authPath, authHandler, nil
}
