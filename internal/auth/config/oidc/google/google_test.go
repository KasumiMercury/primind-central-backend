package google

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadConfigSuccess(t *testing.T) {
	t.Setenv(clientIDEnv, "client-id")
	t.Setenv(clientSecretEnv, "client-secret")
	t.Setenv(redirectURIEnv, "https://example.com/callback")
	t.Setenv(scopesEnv, "openid,email")
	t.Setenv(issuerURLEnv, "https://accounts.google.com")

	cfg, ok, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig returned error: %v", err)
	}

	if !ok {
		t.Fatalf("expected ok=true, got false")
	}

	googleCfg, ok := cfg.(*Config)
	if !ok {
		t.Fatalf("expected *Config, got %T", cfg)
	}

	if googleCfg.ClientID != "client-id" {
		t.Fatalf("ClientID = %s, want client-id", googleCfg.ClientID)
	}

	if googleCfg.ClientSecret != "client-secret" {
		t.Fatalf("ClientSecret = %s, want client-secret", googleCfg.ClientSecret)
	}

	if googleCfg.RedirectURI != "https://example.com/callback" {
		t.Fatalf("RedirectURI = %s, want https://example.com/callback", googleCfg.RedirectURI)
	}

	if len(googleCfg.Scopes) != 2 || googleCfg.Scopes[0] != "openid" || googleCfg.Scopes[1] != "email" {
		t.Fatalf("Scopes = %#v, want [openid email]", googleCfg.Scopes)
	}

	if googleCfg.IssuerURL != "https://accounts.google.com" {
		t.Fatalf("IssuerURL = %s, want https://accounts.google.com", googleCfg.IssuerURL)
	}

	if googleCfg.ProviderID() != "google" {
		t.Fatalf("ProviderID = %s, want google", googleCfg.ProviderID())
	}

	core := googleCfg.Core()
	if core.ClientID != "client-id" {
		t.Fatalf("Core.ClientID = %s, want client-id", core.ClientID)
	}

	if core.ClientSecret != "client-secret" {
		t.Fatalf("Core.ClientSecret = %s, want client-secret", core.ClientSecret)
	}

	if core.RedirectURI != "https://example.com/callback" {
		t.Fatalf("Core.RedirectURI = %s, want https://example.com/callback", core.RedirectURI)
	}

	if len(core.Scopes) != 2 || core.Scopes[0] != "openid" || core.Scopes[1] != "email" {
		t.Fatalf("Core.Scopes = %#v, want [openid email]", core.Scopes)
	}

	if core.IssuerURL != "https://accounts.google.com" {
		t.Fatalf("Core.IssuerURL = %s, want https://accounts.google.com", core.IssuerURL)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	t.Run("missing client id returns ok=false", func(t *testing.T) {
		t.Setenv(clientIDEnv, "")

		cfg, ok, err := loadConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if ok {
			t.Fatalf("expected ok=false when client id is missing")
		}

		if cfg != nil {
			t.Fatalf("expected cfg to be nil")
		}
	})

	t.Run("missing client secret returns error", func(t *testing.T) {
		t.Setenv(clientIDEnv, "client-id")
		t.Setenv(clientSecretEnv, "")
		t.Setenv(redirectURIEnv, "https://example.com/callback")

		_, ok, err := loadConfig()
		if err == nil {
			t.Fatalf("expected error but got nil")
		}

		if !errors.Is(err, ErrGoogleClientSecretMissing) {
			t.Fatalf("expected ErrGoogleClientSecretMissing, got %v", err)
		}

		if ok {
			t.Fatalf("expected ok=false when error occurs")
		}
	})

	t.Run("missing redirect uri returns error", func(t *testing.T) {
		t.Setenv(clientIDEnv, "client-id")
		t.Setenv(clientSecretEnv, "client-secret")
		t.Setenv(redirectURIEnv, "")

		_, ok, err := loadConfig()
		if err == nil {
			t.Fatalf("expected error but got nil")
		}

		if !errors.Is(err, ErrGoogleRedirectURIMissing) {
			t.Fatalf("expected ErrGoogleRedirectURIMissing, got %v", err)
		}

		if ok {
			t.Fatalf("expected ok=false when error occurs")
		}

		if !strings.Contains(err.Error(), "redirect uri") {
			t.Fatalf("expected error to mention redirect uri, got %v", err)
		}
	})
}

func TestGoogleConfigValidateSuccess(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://example.com/callback",
		Scopes:       []string{"openid", "profile"},
		IssuerURL:    "https://accounts.google.com",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestGoogleConfigValidateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *Config
		wantErr error
	}{
		{
			"not URL format issuer",
			&Config{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"openid", "profile"},
				IssuerURL:    "not-a-url",
			},
			ErrGoogleIssuerInvalid,
		},
		{
			"parse failed issuer",
			&Config{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"openid", "profile"},
				IssuerURL:    "http://[::1]:namedport",
			},
			ErrGoogleIssuerInvalid,
		},
		{
			name: "invalid issuer URL",
			cfg: &Config{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"openid", "profile"},
				IssuerURL:    "https://issuer.example.com",
			},
			wantErr: ErrGoogleIssuerInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.cfg.Validate(); err == nil {
				t.Fatalf("expected error but got nil")
			} else if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestGetEnvHelpter(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		want     string
	}{
		{
			"existing env var",
			"TEST_EXISTING_ENV_VAR",
			"some-value",
			"some-value",
		},
		{
			"non-existing env var",
			"TEST_NON_EXISTING_ENV_VAR",
			"",
			"default-value",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envVar, tt.envValue)
			}
			got := getEnv(tt.envVar, "default-value")
			if got != tt.want {
				t.Fatalf("getEnv(%s) = %s, want %s", tt.envVar, got, tt.want)
			}
		})
	}
}

func TestGetEnvRequiredHelper(t *testing.T) {
	tests := []struct {
		name      string
		envVar    string
		envValue  string
		want      string
		expectErr bool
	}{
		{
			"existing env var",
			"TEST_EXISTING_REQUIRED_ENV_VAR",
			"required-value",
			"required-value",
			false,
		},
		{
			"non-existing env var",
			"TEST_NON_EXISTING_REQUIRED_ENV_VAR",
			"",
			"",
			true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envVar, tt.envValue)
			}
			got, err := getEnvRequired(tt.envVar)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}

				if !errors.Is(err, ErrEnvVarRequiredMissing) {
					t.Fatalf("expected ErrEnvVarRequiredMissing, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("getEnvRequired(%s) = %s, want %s", tt.envVar, got, tt.want)
			}
		})
	}
}

func TestGetEnvSliceHelper(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		want     []string
	}{
		{
			"existing env var",
			"TEST_EXISTING_ENV_VAR_SLICE",
			"scope1,scope2,scope3",
			[]string{"scope1", "scope2", "scope3"},
		},
		{
			"non-existing env var",
			"TEST_NON_EXISTING_ENV_VAR_SLICE",
			"",
			[]string{"default-scope1", "default-scope2"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envVar, tt.envValue)
			}
			got := getEnvSlice(tt.envVar, ",", "default-scope1", "default-scope2")
			if len(got) != len(tt.want) {
				t.Fatalf("getEnvSlice(%s) = %v, want %v", tt.envVar, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("getEnvSlice(%s)[%d] = %s, want %s", tt.envVar, i, got[i], tt.want[i])
				}
			}
		})
	}
}
