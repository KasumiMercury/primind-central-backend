package config

import (
	"errors"
	"fmt"

	"github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	_ "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc/google"
	sessioncfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
)

// AuthConfig holds all configuration for the auth module.
type AuthConfig struct {
	Session *sessioncfg.Config
	OIDC    *oidc.Config
}

func Load() (*AuthConfig, error) {
	sessionConfig, err := sessioncfg.Load()
	if err != nil {
		return nil, fmt.Errorf("load session config: %w", err)
	}

	cfg := &AuthConfig{
		Session: sessionConfig,
		OIDC:    nil,
	}

	oidcCfg, err := oidc.Load()
	if err != nil && !errors.Is(err, oidc.ErrNoProvidersConfigured) {
		return nil, fmt.Errorf("load oidc providers: %w", err)
	}

	if oidcCfg != nil {
		cfg.OIDC = oidcCfg
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *AuthConfig) Validate() error {
	if c.Session == nil {
		return ErrSessionConfigMissing
	}

	if err := c.Session.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrSessionConfigMissing, err)
	}

	if c.OIDC != nil {
		if err := c.OIDC.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrOIDCConfigInvalid, err)
		}
	}

	return nil
}
