package config

import (
	"errors"
	"testing"
	"time"

	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	sessioncfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
)

type stubProviderConfig struct {
	id          domainoidc.ProviderID
	core        oidccfg.CoreConfig
	validateErr error
}

func (s stubProviderConfig) ProviderID() domainoidc.ProviderID { return s.id }

func (s stubProviderConfig) Core() oidccfg.CoreConfig { return s.core }

func (s stubProviderConfig) Validate() error { return s.validateErr }

func TestAuthConfigValidateSuccess(t *testing.T) {
	t.Parallel()

	cfg := &AuthConfig{
		Session: &sessioncfg.Config{
			Duration: time.Hour,
			Secret:   "super-secret",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestAuthConfigValidateErrors(t *testing.T) {
	t.Parallel()

	validSession := &sessioncfg.Config{
		Duration: time.Hour,
		Secret:   "super-secret",
	}

	validCore := oidccfg.CoreConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURI:  "https://example.com/callback",
		Scopes:       []string{"openid"},
		IssuerURL:    "https://issuer.example.com",
	}

	tests := []struct {
		name    string
		cfg     *AuthConfig
		wantErr error
	}{
		{
			name:    "missing session config",
			cfg:     &AuthConfig{},
			wantErr: ErrSessionConfigMissing,
		},
		{
			name: "invalid session config",
			cfg: &AuthConfig{
				Session: &sessioncfg.Config{},
			},
			wantErr: sessioncfg.ErrSessionSecretMissing,
		},
		{
			name: "invalid oidc config bubbles up",
			cfg: &AuthConfig{
				Session: validSession,
				OIDC: &oidccfg.Config{
					Providers: map[domainoidc.ProviderID]oidccfg.ProviderConfig{
						domainoidc.ProviderGoogle: stubProviderConfig{
							id:          domainoidc.ProviderGoogle,
							core:        validCore,
							validateErr: assertError,
						},
					},
				},
			},
			wantErr: assertError,
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

func TestLoadSuccessWithoutOIDC(t *testing.T) {
	t.Setenv("SESSION_SECRET", "secret")
	t.Setenv("SESSION_DURATION", "1h")
	clearOIDCEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Session == nil {
		t.Fatalf("expected session config")
	}

	if cfg.OIDC != nil {
		t.Fatalf("expected oidc config to be nil when env vars not set")
	}
}

func TestLoadSuccessWithOIDC(t *testing.T) {
	t.Setenv("SESSION_SECRET", "secret")
	t.Setenv("SESSION_DURATION", "1h")

	t.Setenv("OIDC_GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("OIDC_GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("OIDC_GOOGLE_REDIRECT_URI", "https://example.com/callback")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.OIDC == nil {
		t.Fatalf("expected oidc config to be populated")
	}
}

func TestLoadErrors(t *testing.T) {
	clearOIDCEnv(t)
	t.Run("missing session secret", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "")
		if _, err := Load(); err == nil {
			t.Fatalf("expected error for missing session secret")
		}
	})

	t.Run("incomplete oidc provider", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "secret")
		t.Setenv("SESSION_DURATION", "1h")
		t.Setenv("OIDC_GOOGLE_CLIENT_ID", "client-id")
		t.Setenv("OIDC_GOOGLE_REDIRECT_URI", "https://example.com/callback")
		t.Setenv("OIDC_GOOGLE_CLIENT_SECRET", "")

		if _, err := Load(); err == nil {
			t.Fatalf("expected oidc load error")
		}
	})
}

var assertError = &expectedError{msg: "assert error"}

type expectedError struct {
	msg string
}

func (e *expectedError) Error() string { return e.msg }

func clearOIDCEnv(t *testing.T) {
	t.Helper()

	t.Setenv("OIDC_GOOGLE_CLIENT_ID", "")
	t.Setenv("OIDC_GOOGLE_CLIENT_SECRET", "")
	t.Setenv("OIDC_GOOGLE_REDIRECT_URI", "")
	t.Setenv("OIDC_GOOGLE_SCOPES", "")
	t.Setenv("OIDC_GOOGLE_ISSUER_URL", "")
}
