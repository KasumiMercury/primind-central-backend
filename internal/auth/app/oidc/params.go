package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
)

var (
	ErrOIDCNotConfigured   = errors.New("oidc providers are not configured")
	ErrProviderUnsupported = errors.New("oidc provider is not configured")
)

type OIDCParamsGenerator interface {
	Generate(ctx context.Context, provider domain.ProviderID) (*ParamsResult, error)
}

type OIDCProvider interface {
	ProviderID() domain.ProviderID
	BuildAuthorizationURL(state, nonce, codeChallenge string) string
	ClientID() string
	RedirectURI() string
	Scopes() []string
}

type ParamsResult struct {
	AuthorizationURL string
	State            string
}

type paramsGenerator struct {
	providers map[domain.ProviderID]OIDCProvider
	repo      domain.ParamsRepository
	logger    *slog.Logger
}

func NewParamsGenerator(
	providers map[domain.ProviderID]OIDCProvider,
	repo domain.ParamsRepository,
) OIDCParamsGenerator {
	return &paramsGenerator{
		providers: providers,
		repo:      repo,
		logger:    slog.Default().WithGroup("auth").WithGroup("oidc").WithGroup("params"),
	}
}

func (g *paramsGenerator) Generate(ctx context.Context, provider domain.ProviderID) (*ParamsResult, error) {
	rpProvider, ok := g.providers[provider]
	if !ok {
		g.logger.Warn("oidc params requested for unsupported provider", slog.String("provider", string(provider)))
		return nil, ErrProviderUnsupported
	}

	g.logger.Debug("generating oidc authorization params", slog.String("provider", string(provider)))

	state, err := randomToken()
	if err != nil {
		g.logger.Error("failed to generate state token", slog.String("error", err.Error()))
		return nil, err
	}

	nonce, err := randomToken()
	if err != nil {
		g.logger.Error("failed to generate nonce token", slog.String("error", err.Error()))
		return nil, err
	}

	codeVerifier, err := randomToken()
	if err != nil {
		g.logger.Error("failed to generate code verifier", slog.String("error", err.Error()))
		return nil, err
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	authURL := rpProvider.BuildAuthorizationURL(state, nonce, codeChallenge)

	params, err := domain.NewParams(provider, state, nonce, codeVerifier, time.Now().UTC())
	if err != nil {
		g.logger.Error("failed to build params model", slog.String("error", err.Error()))
		return nil, err
	}

	if err := g.repo.SaveParams(ctx, params); err != nil {
		g.logger.Error("failed to persist oidc params", slog.String("error", err.Error()))
		return nil, err
	}

	g.logger.Debug("generated oidc authorization params", slog.String("provider", string(provider)))

	return &ParamsResult{
		AuthorizationURL: authURL,
		State:            state,
	}, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
