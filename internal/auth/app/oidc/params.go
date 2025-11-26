package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
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
	BuildAuthorizationURL(state, nonce string) string
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
}

func NewParamsGenerator(
	providers map[domain.ProviderID]OIDCProvider,
	repo domain.ParamsRepository,
) OIDCParamsGenerator {
	return &paramsGenerator{
		providers: providers,
		repo:      repo,
	}
}

func (g *paramsGenerator) Generate(ctx context.Context, provider domain.ProviderID) (*ParamsResult, error) {
	rpProvider, ok := g.providers[provider]
	if !ok {
		return nil, ErrProviderUnsupported
	}

	state, err := randomToken()
	if err != nil {
		return nil, err
	}

	nonce, err := randomToken()
	if err != nil {
		return nil, err
	}

	authURL := rpProvider.BuildAuthorizationURL(state, nonce)

	params, err := domain.NewParams(provider, state, nonce, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := g.repo.SaveParams(ctx, params); err != nil {
		return nil, err
	}

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
