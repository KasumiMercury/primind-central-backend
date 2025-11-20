package oidc

import (
	"fmt"
	"net/url"
	"slices"

	"golang.org/x/oauth2"
)

// ProviderID identifies a supported OIDC provider.
type ProviderID string

const (
	ProviderGoogle ProviderID = "google"
)

// ProviderConfig defines what every provider must supply.
type ProviderConfig interface {
	ProviderID() ProviderID
	Core() CoreConfig
	OAuth2Endpoint() oauth2.Endpoint
	Validate() error
}

// CoreConfig holds the OIDC-mandatory settings.
type CoreConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	IssuerURL    string
}

func (c CoreConfig) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}

	if c.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}

	if c.RedirectURI == "" {
		return fmt.Errorf("redirect URI is required")
	}

	parsedURL, err := url.Parse(c.RedirectURI)
	if err != nil {
		return fmt.Errorf("invalid redirect URI: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("redirect URI must include scheme (http:// or https://)")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("redirect URI scheme must be http or https, got: %s", parsedURL.Scheme)
	}

	if len(c.Scopes) == 0 {
		return fmt.Errorf("at least one scope is required")
	}

	if !slices.Contains(c.Scopes, "openid") {
		return fmt.Errorf("'openid' scope is required for OIDC")
	}

	if c.IssuerURL == "" {
		return fmt.Errorf("issuer URL is required")
	}

	parsedIssuer, err := url.Parse(c.IssuerURL)
	if err != nil {
		return fmt.Errorf("invalid issuer URL: %w", err)
	}

	if parsedIssuer.Scheme != "https" {
		return fmt.Errorf("issuer URL must use https, got: %s", parsedIssuer.Scheme)
	}

	return nil
}

// Config holds configured providers keyed by their identifier.
type Config struct {
	Providers map[ProviderID]ProviderConfig
}

func (c *Config) Validate() error {
	if len(c.Providers) == 0 {
		return fmt.Errorf("no oidc providers configured")
	}

	for id, provider := range c.Providers {
		if provider == nil {
			return fmt.Errorf("%s: provider config missing", id)
		}
		if provider.ProviderID() != id {
			return fmt.Errorf("%s: provider identifier mismatch", id)
		}
		if err := provider.Core().Validate(); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		if err := validateEndpoint(provider.OAuth2Endpoint()); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		if err := provider.Validate(); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
	}

	return nil
}

func validateEndpoint(endpoint oauth2.Endpoint) error {
	if endpoint.AuthURL == "" {
		return fmt.Errorf("oauth2 authorization endpoint is required")
	}

	authURL, err := url.Parse(endpoint.AuthURL)
	if err != nil {
		return fmt.Errorf("invalid oauth2 authorization endpoint: %w", err)
	}

	if authURL.Scheme != "https" {
		return fmt.Errorf("oauth2 authorization endpoint must use https, got: %s", authURL.Scheme)
	}

	return nil
}

// ProviderLoader builds a provider configuration.
type ProviderLoader func() (ProviderConfig, bool, error)

var loaders = map[ProviderID]ProviderLoader{}

// RegisterProvider registers a loader for a provider identifier.
func RegisterProvider(id ProviderID, loader ProviderLoader) {
	if loader == nil {
		panic("oidc: loader cannot be nil")
	}
	if _, ok := loaders[id]; ok {
		panic(fmt.Sprintf("oidc: provider %s already registered", id))
	}
	loaders[id] = loader
}

func Load() (*Config, error) {
	if len(loaders) == 0 {
		return nil, nil
	}

	providers := make(map[ProviderID]ProviderConfig)
	for id, loader := range loaders {
		cfg, ok, err := loader()
		if err != nil {
			return nil, fmt.Errorf("%s provider: %w", id, err)
		}
		if ok {
			providers[id] = cfg
		}
	}

	if len(providers) == 0 {
		return nil, nil
	}

	cfg := &Config{Providers: providers}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
