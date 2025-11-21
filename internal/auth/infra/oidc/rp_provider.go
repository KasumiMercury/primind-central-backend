package oidc

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// RPProvider wraps zitadel/oidc RelyingParty
type RPProvider struct {
	rp          rp.RelyingParty
	providerID  oidccfg.ProviderID
	redirectURI string
	scopes      []string
}

// NewRPProvider creates a new RelyingParty for the given provider configuration.
func NewRPProvider(ctx context.Context, providerCfg oidccfg.ProviderConfig) (*RPProvider, error) {
	core := providerCfg.Core()

	relyingParty, err := rp.NewRelyingPartyOIDC(
		ctx,
		core.IssuerURL,
		core.ClientID,
		core.ClientSecret,
		core.RedirectURI,
		core.Scopes,
		rp.WithHTTPClient(httphelper.DefaultHTTPClient),
	)
	if err != nil {
		return nil, fmt.Errorf("create relying party for %s: %w", providerCfg.ProviderID(), err)
	}

	return &RPProvider{
		rp:          relyingParty,
		providerID:  providerCfg.ProviderID(),
		redirectURI: core.RedirectURI,
		scopes:      core.Scopes,
	}, nil
}

func (p *RPProvider) BuildAuthorizationURL(state, nonce string) string {
	baseURL := rp.AuthURL(state, p.rp)

	if nonce != "" {
		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			separator := "&"
			if !strings.Contains(baseURL, "?") {
				separator = "?"
			}
			return baseURL + separator + "nonce=" + url.QueryEscape(nonce)
		}

		query := parsedURL.Query()
		query.Set("nonce", nonce)
		parsedURL.RawQuery = query.Encode()
		return parsedURL.String()
	}

	return baseURL
}

func (p *RPProvider) ProviderID() oidccfg.ProviderID {
	return p.providerID
}

func (p *RPProvider) ClientID() string {
	return p.rp.OAuthConfig().ClientID
}

func (p *RPProvider) RedirectURI() string {
	return p.redirectURI
}

func (p *RPProvider) Scopes() []string {
	return p.scopes
}

func (p *RPProvider) ExchangeToken(ctx context.Context, code string) (*appoidc.IDToken, error) {
	tokens, err := rp.CodeExchange[*oidc.IDTokenClaims](ctx, code, p.rp)
	if err != nil {
		return nil, err
	}

	return &appoidc.IDToken{
		Subject: tokens.IDTokenClaims.Subject,
		Name:    tokens.IDTokenClaims.Name,
		Nonce:   tokens.IDTokenClaims.Nonce,
	}, nil
}
