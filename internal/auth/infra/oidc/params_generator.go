package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/controller/oidc"
	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
)

type ParamsGenerator struct {
	providers map[oidccfg.ProviderID]*RPProvider
	repo      domain.ParamsRepository
}

func NewParamsGenerator(providers map[oidccfg.ProviderID]*RPProvider, repo domain.ParamsRepository) *ParamsGenerator {
	return &ParamsGenerator{
		providers: providers,
		repo:      repo,
	}
}

// Generate creates OIDC authorization parameters for the specified provider.
func (g *ParamsGenerator) Generate(ctx context.Context, provider oidccfg.ProviderID) (*oidc.ParamsResult, error) {
	rpProvider, ok := g.providers[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", oidc.ErrProviderUnsupported, provider)
	}

	state, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	nonce, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	authURL := rpProvider.BuildAuthorizationURL(state, nonce)

	params := domain.Params{
		Provider:  provider,
		State:     state,
		Nonce:     nonce,
		CreatedAt: time.Now().UTC(),
	}

	if err := g.repo.SaveParams(ctx, params); err != nil {
		return nil, fmt.Errorf("persist oidc params: %w", err)
	}

	return &oidc.ParamsResult{
		AuthorizationURL: authURL,
		ClientID:         rpProvider.ClientID(),
		RedirectURI:      rpProvider.RedirectURI(),
		Scope:            strings.Join(rpProvider.Scopes(), " "),
	}, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
