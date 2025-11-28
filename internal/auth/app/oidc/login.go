package oidc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	sessionCfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
)

var (
	ErrInvalidCode  = errors.New("invalid authorization code")
	ErrInvalidState = errors.New("invalid state parameter")
	ErrInvalidNonce = errors.New("nonce validation failed")
)

type OIDCLoginUseCase interface {
	Login(ctx context.Context, req *LoginRequest) (*LoginResult, error)
}

type SessionTokenGenerator interface {
	Generate(session *domain.Session) (string, error)
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
	providers    map[domainoidc.ProviderID]OIDCProviderWithLogin
	paramsRepo   domainoidc.ParamsRepository
	sessionRepo  domain.SessionRepository
	jwtGenerator SessionTokenGenerator
	sessionCfg   *sessionCfg.Config
	logger       *slog.Logger
}

func NewLoginHandler(
	providers map[domainoidc.ProviderID]OIDCProviderWithLogin,
	paramsRepo domainoidc.ParamsRepository,
	sessionRepo domain.SessionRepository,
	jwtGenerator SessionTokenGenerator,
	sessionCfg *sessionCfg.Config,
) OIDCLoginUseCase {
	return &loginHandler{
		providers:    providers,
		paramsRepo:   paramsRepo,
		sessionRepo:  sessionRepo,
		jwtGenerator: jwtGenerator,
		sessionCfg:   sessionCfg,
		logger:       slog.Default().WithGroup("auth").WithGroup("oidc").WithGroup("login"),
	}
}

func (h *loginHandler) Login(ctx context.Context, req *LoginRequest) (*LoginResult, error) {
	rpProvider, ok := h.providers[req.Provider]
	if !ok {
		h.logger.Warn("login attempted with unsupported provider", slog.String("provider", string(req.Provider)))

		return nil, ErrProviderUnsupported
	}

	h.logger.Debug("processing oidc login", slog.String("provider", string(req.Provider)))

	storedParams, err := h.paramsRepo.GetParamsByState(ctx, req.State)
	if err != nil {
		if errors.Is(err, domainoidc.ErrParamsNotFound) {
			h.logger.Warn("state not found during login", slog.String("provider", string(req.Provider)))

			return nil, ErrInvalidState
		}

		h.logger.Error("failed to load stored params", slog.String("error", err.Error()), slog.String("provider", string(req.Provider)))

		return nil, err
	}

	if storedParams.IsExpired(time.Now().UTC()) {
		h.logger.Warn("login attempt with expired params", slog.String("provider", string(req.Provider)))

		return nil, domainoidc.ErrParamsExpired
	}

	if storedParams.Provider() != req.Provider {
		h.logger.Warn("login attempted with mismatched provider", slog.String("provider", string(req.Provider)))

		return nil, ErrInvalidState
	}

	codeVerifier := storedParams.CodeVerifier()

	idToken, err := rpProvider.ExchangeToken(ctx, req.Code, codeVerifier, storedParams.Nonce())
	if err != nil {
		h.logger.Warn("token exchange failed", slog.String("error", err.Error()), slog.String("provider", string(req.Provider)))

		return nil, fmt.Errorf("%w: %v", ErrInvalidCode, err)
	}

	if idToken.Nonce != storedParams.Nonce() {
		h.logger.Warn("nonce validation failed", slog.String("provider", string(req.Provider)))

		return nil, ErrInvalidNonce
	}

	now := time.Now().UTC()
	expiresAt := now.Add(h.sessionCfg.Duration)

	session, err := domain.NewSession(idToken.Subject, now, expiresAt)
	if err != nil {
		h.logger.Error("failed to create session", slog.String("error", err.Error()), slog.String("provider", string(req.Provider)))

		return nil, err
	}

	if err := h.sessionRepo.SaveSession(ctx, session); err != nil {
		h.logger.Error("failed to persist session", slog.String("error", err.Error()), slog.String("provider", string(req.Provider)))

		return nil, err
	}

	sessionToken, err := h.jwtGenerator.Generate(session)
	if err != nil {
		h.logger.Error("failed to generate session token", slog.String("error", err.Error()), slog.String("provider", string(req.Provider)))

		return nil, err
	}

	h.logger.Info("oidc login successful", slog.String("provider", string(req.Provider)))

	return &LoginResult{
		SessionToken: sessionToken,
	}, nil
}
