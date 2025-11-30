package oidc

import (
	"strings"
	"testing"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"golang.org/x/oauth2"
)

func TestRPProviderBuildAuthorizationURLSuccess(t *testing.T) {
	p := &RPProvider{
		oauthConfig: &oauth2.Config{
			ClientID: "client-id",
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://example.com/auth",
			},
			RedirectURL: "https://example.com/redirect",
			Scopes:      []string{"openid", "profile"},
		},
		verifier:    nil,
		providerID:  domainoidc.ProviderGoogle,
		redirectURI: "https://example.com/redirect",
		scopes:      []string{"openid", "profile"},
	}

	url := p.BuildAuthorizationURL("state-xyz", "nonce-abc", "challenge-123")

	if !strings.Contains(url, "state-xyz") {
		t.Fatalf("expected state to be included: %s", url)
	}

	if !strings.Contains(url, "nonce=nonce-abc") {
		t.Fatalf("expected nonce to be included: %s", url)
	}

	if !strings.Contains(url, "code_challenge=challenge-123") {
		t.Fatalf("expected code challenge to be included: %s", url)
	}

	if !strings.Contains(url, "code_challenge_method=S256") {
		t.Fatalf("expected code challenge method S256: %s", url)
	}
}

func TestRPProviderBuildAuthorizationURLError(t *testing.T) {
	p := &RPProvider{
		oauthConfig: &oauth2.Config{
			ClientID: "client-id",
			Endpoint: oauth2.Endpoint{
				AuthURL: "http://example.com/auth%gh", // force url.Parse failure
			},
			RedirectURL: "https://example.com/redirect",
			Scopes:      []string{"openid"},
		},
		verifier:    nil,
		providerID:  domainoidc.ProviderGoogle,
		redirectURI: "https://example.com/redirect",
		scopes:      []string{"openid"},
	}

	url := p.BuildAuthorizationURL("state-123", "nonce-zzz", "cc-111")

	if !strings.Contains(url, "nonce-zzz") || !strings.Contains(url, "cc-111") {
		t.Fatalf("expected fallback URL to include nonce and code challenge: %s", url)
	}
}

func TestRPProviderGettersSuccess(t *testing.T) {
	p := &RPProvider{
		oauthConfig: &oauth2.Config{
			ClientID: "client-id",
		},
		providerID:  domainoidc.ProviderGoogle,
		redirectURI: "https://example.com/r",
		scopes:      []string{"a", "b"},
	}

	if p.ProviderID() != domainoidc.ProviderGoogle {
		t.Fatalf("provider id mismatch")
	}

	if p.ClientID() != "client-id" {
		t.Fatalf("client id mismatch")
	}

	if p.RedirectURI() != "https://example.com/r" {
		t.Fatalf("redirect uri mismatch")
	}

	if got := p.Scopes(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected scopes: %#v", got)
	}
}
