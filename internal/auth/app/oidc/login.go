package oidc

import (
	"context"
	"errors"
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
	ExchangeToken(ctx context.Context, code, codeVerifier string) (*IDToken, error)
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
	}
}

func (h *loginHandler) Login(ctx context.Context, req *LoginRequest) (*LoginResult, error) {
	rpProvider, ok := h.providers[req.Provider]
	if !ok {
		return nil, ErrProviderUnsupported
	}

	storedParams, err := h.paramsRepo.GetParamsByState(ctx, req.State)
	if err != nil {
		if errors.Is(err, domainoidc.ErrParamsNotFound) {
			return nil, ErrInvalidState
		}
		return nil, err
	}

	if storedParams.IsExpired(time.Now().UTC()) {
		return nil, domainoidc.ErrParamsExpired
	}

	if storedParams.Provider() != req.Provider {
		return nil, ErrInvalidState
	}

	codeVerifier := storedParams.CodeVerifier()

	idToken, err := rpProvider.ExchangeToken(ctx, req.Code, codeVerifier)
	if err != nil {
		return nil, ErrInvalidCode
	}

	if idToken.Nonce != storedParams.Nonce() {
		return nil, ErrInvalidNonce
	}

	now := time.Now().UTC()
	expiresAt := now.Add(h.sessionCfg.Duration)

	session, err := domain.NewSession(idToken.Subject, now, expiresAt)
	if err != nil {
		return nil, err
	}

	if err := h.sessionRepo.SaveSession(ctx, session); err != nil {
		return nil, err
	}

	sessionToken, err := h.jwtGenerator.Generate(session)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		SessionToken: sessionToken,
	}, nil
}
