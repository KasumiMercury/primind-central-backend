package oidc

import (
	"fmt"
	"net/url"
	"slices"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
)

// ProviderConfig defines what every provider must supply.
type ProviderConfig interface {
	ProviderID() domainoidc.ProviderID
	Core() CoreConfig
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
		return ErrClientIDMissing
	}

	if c.ClientSecret == "" {
		return ErrClientSecretMissing
	}

	if c.RedirectURI == "" {
		return ErrRedirectURIMissing
	}

	parsedURL, err := url.Parse(c.RedirectURI)
	if err != nil {
		return fmt.Errorf("invalid redirect URI: %w", err)
	}

	if parsedURL.Scheme == "" {
		return ErrRedirectSchemeMissing
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%w, got: %s", ErrRedirectSchemeInvalid, parsedURL.Scheme)
	}

	if len(c.Scopes) == 0 {
		return ErrScopesMissing
	}

	if !slices.Contains(c.Scopes, "openid") {
		return ErrScopeOpenIDRequired
	}

	if c.IssuerURL == "" {
		return ErrIssuerURLMissing
	}

	parsedIssuer, err := url.Parse(c.IssuerURL)
	if err != nil {
		return fmt.Errorf("invalid issuer URL: %w", err)
	}

	if parsedIssuer.Scheme != "https" {
		return fmt.Errorf("%w, got: %s", ErrIssuerURLSchemeInvalid, parsedIssuer.Scheme)
	}

	return nil
}

// Config holds configured providers keyed by their identifier.
type Config struct {
	Providers map[domainoidc.ProviderID]ProviderConfig
}

func (c *Config) Validate() error {
	if len(c.Providers) == 0 {
		return ErrNoOIDCProviders
	}

	for id, provider := range c.Providers {
		if provider == nil {
			return fmt.Errorf("%s: %w", id, ErrProviderConfigNil)
		}

		if provider.ProviderID() != id {
			return fmt.Errorf("%s: %w", id, ErrProviderIDMismatch)
		}

		if err := provider.Core().Validate(); err != nil {
			return fmt.Errorf("%s: %w: %w", id, ErrProviderCoreInvalid, err)
		}

		if err := provider.Validate(); err != nil {
			return fmt.Errorf("%s: %w: %w", id, ErrProviderValidateFail, err)
		}
	}

	return nil
}

// ProviderLoader builds a provider configuration.
type ProviderLoader func() (ProviderConfig, bool, error)

var loaders = map[domainoidc.ProviderID]ProviderLoader{}

// RegisterProvider registers a loader for a provider identifier.
func RegisterProvider(id domainoidc.ProviderID, loader ProviderLoader) {
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
		return nil, ErrNoProvidersConfigured
	}

	providers := make(map[domainoidc.ProviderID]ProviderConfig)

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
		return nil, ErrNoProvidersConfigured
	}

	cfg := &Config{Providers: providers}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
