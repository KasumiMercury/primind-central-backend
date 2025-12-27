package oidc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	sessionCfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity"
	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/clock"
)

type OIDCLoginUseCase interface {
	Login(ctx context.Context, req *LoginRequest) (*LoginResult, error)
}

type UserWithOIDCIdentityRepository interface {
	SaveUserWithOIDCIdentity(ctx context.Context, user *user.User, identity *oidcidentity.OIDCIdentity) error
}

type SessionTokenGenerator interface {
	Generate(session *domain.Session, user *user.User) (string, error)
}

type IDToken struct {
	Subject string
	Name    string
	Nonce   string
}

type OIDCProviderWithLogin interface {
	OIDCProvider
	ExchangeToken(ctx context.Context, code, codeVerifier, nonce string) (*IDToken, error)
}

type LoginRequest struct {
	Provider domainoidc.ProviderID
	Code     string
	State    string
}

type LoginResult struct {
	SessionToken string
}

type loginHandler struct {
	providers        map[domainoidc.ProviderID]OIDCProviderWithLogin
	paramsRepo       domainoidc.ParamsRepository
	sessionRepo      domain.SessionRepository
	userRepo         user.UserRepository
	oidcIdentityRepo oidcidentity.OIDCIdentityRepository
	userIdentityRepo UserWithOIDCIdentityRepository
	jwtGenerator     SessionTokenGenerator
	sessionCfg       *sessionCfg.Config
	clock            clock.Clock
	logger           *slog.Logger
}

func NewLoginHandler(
	providers map[domainoidc.ProviderID]OIDCProviderWithLogin,
	paramsRepo domainoidc.ParamsRepository,
	sessionRepo domain.SessionRepository,
	userRepo user.UserRepository,
	oidcIdentityRepo oidcidentity.OIDCIdentityRepository,
	userIdentityRepo UserWithOIDCIdentityRepository,
	jwtGenerator SessionTokenGenerator,
	sessionCfg *sessionCfg.Config,
) OIDCLoginUseCase {
	return newLoginHandler(
		providers,
		paramsRepo,
		sessionRepo,
		userRepo,
		oidcIdentityRepo,
		userIdentityRepo,
		jwtGenerator,
		sessionCfg,
		&clock.RealClock{},
	)
}

func NewLoginHandlerWithClock(
	providers map[domainoidc.ProviderID]OIDCProviderWithLogin,
	paramsRepo domainoidc.ParamsRepository,
	sessionRepo domain.SessionRepository,
	userRepo user.UserRepository,
	oidcIdentityRepo oidcidentity.OIDCIdentityRepository,
	userIdentityRepo UserWithOIDCIdentityRepository,
	jwtGenerator SessionTokenGenerator,
	sessionCfg *sessionCfg.Config,
	clk clock.Clock,
) OIDCLoginUseCase {
	return newLoginHandler(
		providers,
		paramsRepo,
		sessionRepo,
		userRepo,
		oidcIdentityRepo,
		userIdentityRepo,
		jwtGenerator,
		sessionCfg,
		clk,
	)
}

func newLoginHandler(
	providers map[domainoidc.ProviderID]OIDCProviderWithLogin,
	paramsRepo domainoidc.ParamsRepository,
	sessionRepo domain.SessionRepository,
	userRepo user.UserRepository,
	oidcIdentityRepo oidcidentity.OIDCIdentityRepository,
	userIdentityRepo UserWithOIDCIdentityRepository,
	jwtGenerator SessionTokenGenerator,
	sessionCfg *sessionCfg.Config,
	clk clock.Clock,
) OIDCLoginUseCase {
	return &loginHandler{
		providers:        providers,
		paramsRepo:       paramsRepo,
		sessionRepo:      sessionRepo,
		userRepo:         userRepo,
		oidcIdentityRepo: oidcIdentityRepo,
		userIdentityRepo: userIdentityRepo,
		jwtGenerator:     jwtGenerator,
		sessionCfg:       sessionCfg,
		clock:            clk,
		logger:           slog.Default().With(slog.String("module", "auth")).WithGroup("auth").WithGroup("oidc").WithGroup("login"),
	}
}

func (h *loginHandler) Login(ctx context.Context, req *LoginRequest) (*LoginResult, error) {
	rpProvider, ok := h.providers[req.Provider]
	if !ok {
		h.logger.Warn("login attempted with unsupported provider", slog.String("provider", string(req.Provider)))

		return nil, ErrOIDCProviderUnsupported
	}

	h.logger.Debug("processing oidc login", slog.String("provider", string(req.Provider)))

	storedParams, err := h.loadAndValidateParams(ctx, req)
	if err != nil {
		return nil, err
	}

	idToken, err := h.exchangeAndValidateIDToken(ctx, rpProvider, req, storedParams)
	if err != nil {
		return nil, err
	}

	userID, targetUser, err := h.resolveUser(ctx, req.Provider, idToken)
	if err != nil {
		return nil, err
	}

	now := h.clock.Now()
	expiresAt := now.Add(h.sessionCfg.Duration)

	session, err := domain.NewSession(userID, now, expiresAt)
	if err != nil {
		h.logger.Error("failed to create session", slog.String("error", err.Error()))

		return nil, err
	}

	if err := h.sessionRepo.SaveSession(ctx, session); err != nil {
		h.logger.Error("failed to persist session", slog.String("error", err.Error()))

		return nil, err
	}

	sessionToken, err := h.jwtGenerator.Generate(session, targetUser)
	if err != nil {
		h.logger.Error("failed to generate session token", slog.String("error", err.Error()), slog.String("provider", string(req.Provider)))

		return nil, err
	}

	h.logger.Info("oidc login successful", slog.String("provider", string(req.Provider)))

	return &LoginResult{
		SessionToken: sessionToken,
	}, nil
}

func (h *loginHandler) loadAndValidateParams(ctx context.Context, req *LoginRequest) (*domainoidc.Params, error) {
	storedParams, err := h.paramsRepo.GetParamsByState(ctx, req.State)
	if err != nil {
		if errors.Is(err, domainoidc.ErrParamsNotFound) {
			h.logger.Warn("state not found during login", slog.String("provider", string(req.Provider)))

			return nil, ErrStateInvalid
		}

		h.logger.Error("failed to load stored params", slog.String("error", err.Error()), slog.String("provider", string(req.Provider)))

		return nil, err
	}

	if storedParams.IsExpired(h.clock.Now()) {
		h.logger.Warn("login attempt with expired params", slog.String("provider", string(req.Provider)))

		return nil, domainoidc.ErrParamsExpired
	}

	if storedParams.Provider() != req.Provider {
		h.logger.Warn("login attempted with mismatched provider", slog.String("provider", string(req.Provider)))

		return nil, ErrStateInvalid
	}

	return storedParams, nil
}

func (h *loginHandler) exchangeAndValidateIDToken(
	ctx context.Context,
	rpProvider OIDCProviderWithLogin,
	req *LoginRequest,
	params *domainoidc.Params,
) (*IDToken, error) {
	idToken, err := rpProvider.ExchangeToken(ctx, req.Code, params.CodeVerifier(), params.Nonce())
	if err != nil {
		h.logger.Warn("token exchange failed", slog.String("error", err.Error()), slog.String("provider", string(req.Provider)))

		return nil, fmt.Errorf("%w: %v", ErrCodeInvalid, err)
	}

	if idToken.Nonce != params.Nonce() {
		h.logger.Warn("nonce validation failed", slog.String("provider", string(req.Provider)))

		return nil, ErrNonceInvalid
	}

	return idToken, nil
}

func (h *loginHandler) resolveUser(
	ctx context.Context,
	provider domainoidc.ProviderID,
	idToken *IDToken,
) (user.ID, *user.User, error) {
	oidcIdentity, err := h.oidcIdentityRepo.GetOIDCIdentityByProviderSubject(ctx, provider, idToken.Subject)
	if err != nil && !errors.Is(err, oidcidentity.ErrOIDCIdentityNotFound) {
		h.logger.Error("failed to lookup oidc identity", slog.String("error", err.Error()))

		return user.ID{}, nil, err
	}

	if oidcIdentity == nil {
		return h.createUserAndIdentity(ctx, provider, idToken.Subject)
	}

	h.logger.Debug("existing user found for oidc login", slog.String("provider", string(provider)))

	existingUser, err := h.userRepo.GetUserByID(ctx, oidcIdentity.UserID())
	if err != nil {
		h.logger.Error("failed to load user", slog.String("error", err.Error()))

		return user.ID{}, nil, err
	}

	return existingUser.ID(), existingUser, nil
}

func (h *loginHandler) createUserAndIdentity(
	ctx context.Context,
	provider domainoidc.ProviderID,
	subject string,
) (user.ID, *user.User, error) {
	h.logger.Debug("creating new user for oidc login", slog.String("provider", string(provider)))

	newUser, err := user.CreateUserWithRandomColor()
	if err != nil {
		h.logger.Error("failed to generate user ID", slog.String("error", err.Error()))

		return user.ID{}, nil, err
	}

	newIdentity, err := oidcidentity.NewOIDCIdentity(newUser.ID(), provider, subject)
	if err != nil {
		h.logger.Error("failed to create oidc identity", slog.String("error", err.Error()))

		return user.ID{}, nil, err
	}

	if err := h.userIdentityRepo.SaveUserWithOIDCIdentity(ctx, newUser, newIdentity); err != nil {
		h.logger.Error("failed to persist user and oidc identity", slog.String("error", err.Error()))

		return user.ID{}, nil, err
	}

	return newUser.ID(), newUser, nil
}
