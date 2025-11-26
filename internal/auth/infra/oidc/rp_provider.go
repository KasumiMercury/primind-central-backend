package oidc

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// RPProvider wraps zitadel/oidc RelyingParty
type RPProvider struct {
	rp          rp.RelyingParty
	providerID  domainoidc.ProviderID
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

func (p *RPProvider) BuildAuthorizationURL(state, nonce, codeChallenge string) string {
	baseURL := rp.AuthURL(state, p.rp)

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		// Fallback to string concatenation if URL parsing fails
		separator := "&"
		if !strings.Contains(baseURL, "?") {
			separator = "?"
		}
		result := baseURL
		if nonce != "" {
			result += separator + "nonce=" + url.QueryEscape(nonce)
			separator = "&"
		}
		if codeChallenge != "" {
			result += separator + "code_challenge=" + url.QueryEscape(codeChallenge)
			result += "&code_challenge_method=S256"
		}
		return result
	}

	query := parsedURL.Query()
	if nonce != "" {
		query.Set("nonce", nonce)
	}
	if codeChallenge != "" {
		query.Set("code_challenge", codeChallenge)
		query.Set("code_challenge_method", "S256")
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

func (p *RPProvider) ProviderID() domainoidc.ProviderID {
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

func (p *RPProvider) ExchangeToken(ctx context.Context, code, codeVerifier string) (*appoidc.IDToken, error) {
	tokens, err := rp.CodeExchange[*oidc.IDTokenClaims](
		ctx,
		code,
		p.rp,
		rp.WithCodeVerifier(codeVerifier),
	)
	if err != nil {
		return nil, err
	}

	return &appoidc.IDToken{
		Subject: tokens.IDTokenClaims.Subject,
		Name:    tokens.IDTokenClaims.Name,
		Nonce:   tokens.IDTokenClaims.Nonce,
	}, nil
}
