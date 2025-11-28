package session

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	sessionSecretEnv   = "SESSION_SECRET"
	sessionDurationEnv = "SESSION_DURATION"

	defaultSessionDuration = 24 * time.Hour
)

var (
	ErrSessionSecretMissing   = errors.New("session secret is required")
	ErrSessionDurationInvalid = errors.New("session duration must be positive")
)

// Config contains session management settings.
type Config struct {
	Duration time.Duration
	Secret   string
}

func Load() (*Config, error) {
	secret, err := getEnvRequired(sessionSecretEnv)
	if err != nil {
		return nil, err
	}

	return &Config{
		Duration: getEnvDuration(sessionDurationEnv, defaultSessionDuration),
		Secret:   secret,
	}, nil
}

func (c *Config) Validate() error {
	if c.Secret == "" {
		return ErrSessionSecretMissing
	}

	if c.Duration <= 0 {
		return fmt.Errorf("%w, got: %v", ErrSessionDurationInvalid, c.Duration)
	}

	return nil
}

func getEnvRequired(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("%w: %s", ErrSessionSecretMissing, key)
	}

	return val, nil
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}

	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}

	return d
}
