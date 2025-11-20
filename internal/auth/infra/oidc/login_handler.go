package oidc

import (
	"context"
	"errors"
	"fmt"
	"time"

	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/controller/oidc"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/jwt"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
)

type LoginHandler struct {
	providers       map[oidccfg.ProviderID]*RPProvider
	paramsRepo      domainoidc.ParamsRepository
	sessionRepo     domain.SessionRepository
	jwtGenerator    *jwt.Generator
	sessionDuration time.Duration
}

func NewLoginHandler(
	providers map[oidccfg.ProviderID]*RPProvider,
	paramsRepo domainoidc.ParamsRepository,
	sessionRepo domain.SessionRepository,
	jwtGenerator *jwt.Generator,
	sessionDuration time.Duration,
) *LoginHandler {
	return &LoginHandler{
		providers:       providers,
		paramsRepo:      paramsRepo,
		sessionRepo:     sessionRepo,
		jwtGenerator:    jwtGenerator,
		sessionDuration: sessionDuration,
	}
}

func (h *LoginHandler) Login(ctx context.Context, req *oidc.LoginRequest) (*oidc.LoginResult, error) {
	rpProvider, ok := h.providers[req.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", oidc.ErrProviderUnsupported, req.Provider)
	}

	storedParams, err := h.paramsRepo.GetParamsByState(ctx, req.State)
	if err != nil {
		if errors.Is(err, repository.ErrParamsNotFound) {
			return nil, fmt.Errorf("%w: %v", oidc.ErrInvalidState, err)
		}
		return nil, fmt.Errorf("get params by state: %w", err)
	}

	if storedParams.Provider != req.Provider {
		return nil, fmt.Errorf("%w: provider mismatch", oidc.ErrInvalidState)
	}

	tokens, err := rpProvider.ExchangeToken(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", oidc.ErrInvalidCode, err)
	}

	if tokens.IDTokenClaims.Nonce != storedParams.Nonce {
		return nil, oidc.ErrInvalidNonce
	}

	sub := tokens.IDTokenClaims.Subject
	name := tokens.IDTokenClaims.Name

	sessionToken, err := h.jwtGenerator.Generate(sub, name)
	if err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(h.sessionDuration)

	session := &domain.Session{
		ID:        sessionToken,
		UserID:    sub,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	if err := h.sessionRepo.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	return &oidc.LoginResult{
		SessionID: session.ID,
		UserID:    session.UserID,
		CreatedAt: session.CreatedAt.Unix(),
		ExpiresAt: session.ExpiresAt.Unix(),
	}, nil
}
