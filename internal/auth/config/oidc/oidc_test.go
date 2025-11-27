package oidc

import (
	"errors"
	"strings"
	"testing"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
)

type stubProvider struct {
	id          domainoidc.ProviderID
	coreCfg     CoreConfig
	validateErr error
}

func (s stubProvider) ProviderID() domainoidc.ProviderID { return s.id }

func (s stubProvider) Core() CoreConfig { return s.coreCfg }

func (s stubProvider) Validate() error { return s.validateErr }

func TestCoreConfigValidateSuccess(t *testing.T) {
	t.Parallel()

	cfg := CoreConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://example.com/callback",
		Scopes:       []string{"openid", "profile"},
		IssuerURL:    "https://issuer.example.com",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestCoreConfigValidateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     CoreConfig
		wantErr error
	}{
		{
			name:    "missing client id",
			cfg:     CoreConfig{},
			wantErr: ErrClientIDMissing,
		},
		{
			name: "missing client secret",
			cfg: CoreConfig{
				ClientID: "client-id",
			},
			wantErr: ErrClientSecretMissing,
		},
		{
			name: "missing redirect uri",
			cfg: CoreConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
			},
			wantErr: ErrRedirectURIMissing,
		},
		{
			name: "redirect scheme missing",
			cfg: CoreConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURI:  "example.com/callback",
			},
			wantErr: ErrRedirectSchemeMissing,
		},
		{
			name: "redirect scheme not http/https",
			cfg: CoreConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURI:  "ftp://example.com/callback",
			},
			wantErr: ErrRedirectSchemeInvalid,
		},
		{
			name: "missing scopes",
			cfg: CoreConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURI:  "https://example.com/callback",
			},
			wantErr: ErrScopesMissing,
		},
		{
			name: "missing openid scope",
			cfg: CoreConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"profile"},
			},
			wantErr: ErrScopeOpenIDRequired,
		},
		{
			name: "missing issuer",
			cfg: CoreConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"openid"},
			},
			wantErr: ErrIssuerURLMissing,
		},
		{
			name: "issuer must be https",
			cfg: CoreConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"openid"},
				IssuerURL:    "http://issuer.example.com",
			},
			wantErr: ErrIssuerURLSchemeInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestConfigValidateSuccess(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Providers: map[domainoidc.ProviderID]ProviderConfig{
			domainoidc.ProviderGoogle: stubProvider{
				id:      domainoidc.ProviderGoogle,
				coreCfg: validCoreConfig(),
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestConfigValidateErrors(t *testing.T) {
	t.Parallel()

	validCore := validCoreConfig()

	tests := []struct {
		name    string
		cfg     *Config
		wantErr error
	}{
		{
			name:    "no providers",
			cfg:     &Config{Providers: map[domainoidc.ProviderID]ProviderConfig{}},
			wantErr: ErrNoOIDCProviders,
		},
		{
			name: "nil provider",
			cfg: &Config{
				Providers: map[domainoidc.ProviderID]ProviderConfig{
					domainoidc.ProviderGoogle: nil,
				},
			},
			wantErr: ErrProviderConfigNil,
		},
		{
			name: "provider id mismatch",
			cfg: &Config{
				Providers: map[domainoidc.ProviderID]ProviderConfig{
					domainoidc.ProviderGoogle: stubProvider{
						id:      "other",
						coreCfg: validCore,
					},
				},
			},
			wantErr: ErrProviderIDMismatch,
		},
		{
			name: "invalid core config",
			cfg: &Config{
				Providers: map[domainoidc.ProviderID]ProviderConfig{
					domainoidc.ProviderGoogle: stubProvider{
						id:      domainoidc.ProviderGoogle,
						coreCfg: CoreConfig{},
					},
				},
			},
			wantErr: ErrProviderCoreInvalid,
		},
		{
			name: "provider validate error",
			cfg: &Config{
				Providers: map[domainoidc.ProviderID]ProviderConfig{
					domainoidc.ProviderGoogle: stubProvider{
						id:          domainoidc.ProviderGoogle,
						coreCfg:     validCore,
						validateErr: errors.New("bad provider"),
					},
				},
			},
			wantErr: ErrProviderValidateFail,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestLoadProviders(t *testing.T) {
	// Do not run in parallel; mutates package-level loaders.
	originalLoaders := loaders
	defer func() { loaders = originalLoaders }()

	stub := stubProvider{
		id:      domainoidc.ProviderGoogle,
		coreCfg: validCoreConfig(),
	}

	loaders = map[domainoidc.ProviderID]ProviderLoader{
		domainoidc.ProviderGoogle: func() (ProviderConfig, bool, error) {
			return stub, true, nil
		},
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected config but got nil")
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(cfg.Providers))
	}
	if _, ok := cfg.Providers[domainoidc.ProviderGoogle]; !ok {
		t.Fatalf("expected google provider to be present")
	}
}

func TestLoadProvidersErrors(t *testing.T) {
	// Do not run in parallel; mutates package-level loaders.
	originalLoaders := loaders
	defer func() { loaders = originalLoaders }()

	tests := []struct {
		name      string
		setup     func()
		wantErr   bool
		wantNil   bool
		wantMsg   string
		expectNil bool
	}{
		{
			name: "loader returns error",
			setup: func() {
				loaders = map[domainoidc.ProviderID]ProviderLoader{
					domainoidc.ProviderGoogle: func() (ProviderConfig, bool, error) {
						return nil, false, errors.New("loader failed")
					},
				}
			},
			wantErr: true,
			wantMsg: "loader failed",
		},
		{
			name: "no loaders registered returns nil",
			setup: func() {
				loaders = map[domainoidc.ProviderID]ProviderLoader{}
			},
			wantNil: true,
		},
		{
			name: "loader returns ok=false results in nil",
			setup: func() {
				loaders = map[domainoidc.ProviderID]ProviderLoader{
					domainoidc.ProviderGoogle: func() (ProviderConfig, bool, error) {
						return nil, false, nil
					},
				}
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			cfg, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
					t.Fatalf("expected error to contain %q, got %v", tt.wantMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if cfg != nil {
					t.Fatalf("expected nil config, got %#v", cfg)
				}
			} else if cfg == nil {
				t.Fatalf("expected config, got nil")
			}
		})
	}
}

func validCoreConfig() CoreConfig {
	return CoreConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://example.com/callback",
		Scopes:       []string{"openid", "profile"},
		IssuerURL:    "https://issuer.example.com",
	}
}
