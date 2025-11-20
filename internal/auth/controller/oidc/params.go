package oidc

import (
	"context"
	"errors"

	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
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
