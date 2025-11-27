package oidc

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// RPProvider wraps go-oidc and oauth2 helpers for a single provider.
type RPProvider struct {
	oauthConfig *oauth2.Config
	verifier    *oidc.IDTokenVerifier
	providerID  domainoidc.ProviderID
	redirectURI string
	scopes      []string
}

// NewRPProvider creates a new relying party backed by go-oidc.
func NewRPProvider(ctx context.Context, providerCfg oidccfg.ProviderConfig) (*RPProvider, error) {
	core := providerCfg.Core()

	oidcProvider, err := oidc.NewProvider(ctx, core.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider %s: %w", providerCfg.ProviderID(), err)
	}

	verifier := oidcProvider.Verifier(&oidc.Config{
		ClientID: core.ClientID,
	})

	oauthConfig := &oauth2.Config{
		ClientID:     core.ClientID,
		ClientSecret: core.ClientSecret,
		Endpoint:     oidcProvider.Endpoint(),
		RedirectURL:  core.RedirectURI,
		Scopes:       core.Scopes,
	}

	return &RPProvider{
		oauthConfig: oauthConfig,
		verifier:    verifier,
		providerID:  providerCfg.ProviderID(),
		redirectURI: core.RedirectURI,
		scopes:      core.Scopes,
	}, nil
}

func (p *RPProvider) BuildAuthorizationURL(state, nonce, codeChallenge string) string {
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("nonce", nonce),
	}
	if codeChallenge != "" {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}
	baseURL := p.oauthConfig.AuthCodeURL(state, opts...)

	// Safety: add nonce/code_challenge if the provider stripped it.
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		separator := "&"
		if !strings.Contains(baseURL, "?") {
			separator = "?"
		}
		result := baseURL
		if nonce != "" && !strings.Contains(baseURL, "nonce=") {
			result += separator + "nonce=" + url.QueryEscape(nonce)
			separator = "&"
		}
		if codeChallenge != "" && !strings.Contains(baseURL, "code_challenge=") {
			result += separator + "code_challenge=" + url.QueryEscape(codeChallenge)
			result += "&code_challenge_method=S256"
		}
		return result
	}

	query := parsedURL.Query()
	if nonce != "" && query.Get("nonce") == "" {
		query.Set("nonce", nonce)
	}
	if codeChallenge != "" && query.Get("code_challenge") == "" {
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
	return p.oauthConfig.ClientID
}

func (p *RPProvider) RedirectURI() string {
	return p.redirectURI
}

func (p *RPProvider) Scopes() []string {
	return p.scopes
}

func (p *RPProvider) ExchangeToken(ctx context.Context, code, codeVerifier, nonce string) (*appoidc.IDToken, error) {
	token, err := p.oauthConfig.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, fmt.Errorf("token exchange failed: id_token missing")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("id_token verification failed: %w", err)
	}

	if nonce != "" && idToken.Nonce != nonce {
		return nil, fmt.Errorf("nonce mismatch")
	}

	var claims struct {
		Sub  string `json:"sub"`
		Name string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode id_token claims: %w", err)
	}

	return &appoidc.IDToken{
		Subject: claims.Sub,
		Name:    claims.Name,
		Nonce:   idToken.Nonce,
	}, nil
}
