package google

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
)

const (
	clientIDEnv     = "OIDC_GOOGLE_CLIENT_ID"
	clientSecretEnv = "OIDC_GOOGLE_CLIENT_SECRET"
	redirectURIEnv  = "OIDC_GOOGLE_REDIRECT_URI"
	scopesEnv       = "OIDC_GOOGLE_SCOPES"
	issuerURLEnv    = "OIDC_GOOGLE_ISSUER_URL"
)

var (
	ErrGoogleClientSecretMissing = errors.New("google oidc client secret missing")
	ErrGoogleRedirectURIMissing  = errors.New("google oidc redirect uri missing")
	ErrGoogleIssuerInvalid       = errors.New("issuer URL host should contain 'google.com'")
)

func init() {
	oidc.RegisterProvider(domainoidc.ProviderGoogle, loadConfig)
}

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	IssuerURL    string
}

var _ oidc.ProviderConfig = (*Config)(nil)

func loadConfig() (oidc.ProviderConfig, bool, error) {
	clientID := os.Getenv(clientIDEnv)
	if clientID == "" {
		return nil, false, nil
	}

	clientSecret, err := getEnvRequired(clientSecretEnv)
	if err != nil {
		return nil, false, fmt.Errorf("client secret: %w", err)
	}

	redirectURI, err := getEnvRequired(redirectURIEnv)
	if err != nil {
		return nil, false, fmt.Errorf("redirect uri: %w", err)
	}

	cfg := &Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes:       getEnvSlice(scopesEnv, ",", "openid", "profile"),
		IssuerURL:    getEnv(issuerURLEnv, "https://accounts.google.com"),
	}

	return cfg, true, nil
}

func (c *Config) ProviderID() domainoidc.ProviderID {
	return domainoidc.ProviderGoogle
}

func (c *Config) Core() oidc.CoreConfig {
	return oidc.CoreConfig{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURI:  c.RedirectURI,
		Scopes:       c.Scopes,
		IssuerURL:    c.IssuerURL,
	}
}

func (c *Config) Validate() error {
	parsedIssuer, err := url.Parse(c.IssuerURL)
	if err != nil {
		return fmt.Errorf("invalid issuer URL: %w", err)
	}

	if !strings.Contains(parsedIssuer.Host, "google.com") {
		return fmt.Errorf("%w, got: %s", ErrGoogleIssuerInvalid, parsedIssuer.Host)
	}

	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvRequired(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		switch key {
		case clientSecretEnv:
			return "", fmt.Errorf("%w: %s", ErrGoogleClientSecretMissing, key)
		case redirectURIEnv:
			return "", fmt.Errorf("%w: %s", ErrGoogleRedirectURIMissing, key)
		default:
			return "", fmt.Errorf("required environment variable %s not set", key)
		}
	}
	return val, nil
}

func getEnvSlice(key, sep string, defaults ...string) []string {
	val := os.Getenv(key)
	if val == "" {
		return defaults
	}
	parts := strings.Split(val, sep)

	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}
