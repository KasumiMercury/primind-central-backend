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

		if !strings.Contains(err.Error(), clientSecretEnv) {
			t.Fatalf("expected error to mention %s, got %v", clientSecretEnv, err)
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

	cfg := &Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "https://example.com/callback",
		Scopes:       []string{"openid", "profile"},
		IssuerURL:    "https://issuer.example.com",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error but got nil")
	} else if !errors.Is(err, ErrGoogleIssuerInvalid) {
		t.Fatalf("expected ErrGoogleIssuerInvalid, got %v", err)
	}
}
