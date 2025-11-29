package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	authServiceURLEnv     = "AUTH_SERVICE_URL"
	defaultAuthServiceURL = "http://localhost:8080"
)

var (
	ErrAuthServiceURLInvalid = errors.New("auth service URL is invalid")
)

type Config struct {
	AuthServiceURL string
}

func Load() (*Config, error) {
	authServiceURL := getEnv(authServiceURLEnv, defaultAuthServiceURL)

	cfg := &Config{
		AuthServiceURL: authServiceURL,
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: config is nil", ErrAuthServiceURLInvalid)
	}

	if c.AuthServiceURL == "" {
		return fmt.Errorf("%w: auth service URL is empty", ErrAuthServiceURLInvalid)
	}

	parsedURL, err := url.Parse(c.AuthServiceURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAuthServiceURLInvalid, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https, got: %s",
			ErrAuthServiceURLInvalid, parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("%w: host is empty", ErrAuthServiceURLInvalid)
	}

	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return strings.TrimSpace(val)
	}

	return defaultVal
}
