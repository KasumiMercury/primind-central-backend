package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	authServiceURLEnv       = "AUTH_SERVICE_URL"
	deviceServiceURLEnv     = "DEVICE_SERVICE_URL"
	defaultAuthServiceURL   = "http://localhost:8080"
	defaultDeviceServiceURL = "http://localhost:8080"
)

type Config struct {
	AuthServiceURL   string
	DeviceServiceURL string
}

func Load() (*Config, error) {
	authServiceURL := getEnv(authServiceURLEnv, defaultAuthServiceURL)
	deviceServiceURL := getEnv(deviceServiceURLEnv, defaultDeviceServiceURL)

	cfg := &Config{
		AuthServiceURL:   authServiceURL,
		DeviceServiceURL: deviceServiceURL,
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: config is nil", ErrAuthServiceURLInvalid)
	}

	if err := c.validateAuthServiceURL(); err != nil {
		return err
	}

	if err := c.validateDeviceServiceURL(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateAuthServiceURL() error {
	return validateServiceURL(c.AuthServiceURL, ErrAuthServiceURLInvalid)
}

func (c *Config) validateDeviceServiceURL() error {
	return validateServiceURL(c.DeviceServiceURL, ErrDeviceServiceURLInvalid)
}

func validateServiceURL(urlStr string, baseErr error) error {
	if urlStr == "" {
		return fmt.Errorf("%w: service URL is empty", baseErr)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("%w: %v", baseErr, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https, got: %s",
			baseErr, parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("%w: host is empty", baseErr)
	}

	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return strings.TrimSpace(val)
	}

	return defaultVal
}
