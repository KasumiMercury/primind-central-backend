package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"golang.org/x/oauth2"
)

var (
	ErrOIDCNotConfigured   = errors.New("oidc providers are not configured")
	ErrProviderUnsupported = errors.New("oidc provider is not configured")
)

type OIDCParamsGenerator interface {
	Generate(ctx context.Context, provider oidccfg.ProviderID) (*ParamsResult, error)
}

type ParamsResult struct {
	AuthorizationURL string
	ClientID         string
	RedirectURI      string
	Scope            string
}

type ParamsUseCase struct {
	cfg  *oidccfg.Config
	repo domain.ParamsRepository
}

func NewParamsUseCase(cfg *oidccfg.Config, repo domain.ParamsRepository) *ParamsUseCase {
	return &ParamsUseCase{
		cfg:  cfg,
		repo: repo,
	}
}

func (u *ParamsUseCase) Generate(ctx context.Context, provider oidccfg.ProviderID) (*ParamsResult, error) {
	if u.cfg == nil || len(u.cfg.Providers) == 0 {
		return nil, ErrOIDCNotConfigured
	}

	providerCfg, ok := u.cfg.Providers[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnsupported, provider)
	}

	core := providerCfg.Core()

	state, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	nonce, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	authURL, err := buildAuthorizationURL(providerCfg, core, state, nonce)
	if err != nil {
		return nil, err
	}

	params := domain.Params{
		Provider:  provider,
		State:     state,
		Nonce:     nonce,
		CreatedAt: time.Now().UTC(),
	}

	if err := u.repo.SaveParams(ctx, params); err != nil {
		return nil, fmt.Errorf("persist oidc params: %w", err)
	}

	return &ParamsResult{
		AuthorizationURL: authURL,
		ClientID:         core.ClientID,
		RedirectURI:      core.RedirectURI,
		Scope:            strings.Join(core.Scopes, " "),
	}, nil
}

func buildAuthorizationURL(providerCfg oidccfg.ProviderConfig, core oidccfg.CoreConfig, state, nonce string) (string, error) {
	endpoint := providerCfg.OAuth2Endpoint()
	if endpoint.AuthURL == "" {
		return "", fmt.Errorf("oauth2 authorization endpoint is required")
	}

	oauth2Cfg := oauth2.Config{
		ClientID:     core.ClientID,
		ClientSecret: core.ClientSecret,
		RedirectURL:  core.RedirectURI,
		Scopes:       core.Scopes,
		Endpoint:     endpoint,
	}

	return oauth2Cfg.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce)), nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
