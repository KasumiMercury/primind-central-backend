package config

import (
	"errors"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name              string
		envAuthServiceURL string
		expected          *Config
	}{
		{
			"valid AuthServiceURL",
			"https://auth.example.com",
			&Config{
				AuthServiceURL: "https://auth.example.com",
			},
		},
		{
			"default AuthServiceURL",
			"",
			&Config{
				AuthServiceURL: "http://localhost:8080",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AUTH_SERVICE_URL", tt.envAuthServiceURL)

			got, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}

			if got.AuthServiceURL != tt.expected.AuthServiceURL {
				t.Fatalf("Load() AuthServiceURL = %s, want %s", got.AuthServiceURL, tt.expected.AuthServiceURL)
			}
		})
	}
}

func TestLoadError(t *testing.T) {
	tests := []struct {
		name              string
		envAuthServiceURL string
		expectedErr       error
	}{
		{
			"invalid AuthServiceURL",
			"ftp://auth.example.com",
			ErrAuthServiceURLInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AUTH_SERVICE_URL", tt.envAuthServiceURL)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() error = nil, want %v", tt.expectedErr)
			}

			if errors.Is(err, tt.expectedErr) == false {
				t.Fatalf("Load() error = %v, want %v", err, tt.expectedErr)
			}
		})
	}
}

func TestValidateSuccess(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			"valid config",
			Config{
				AuthServiceURL: "https://auth.example.com",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestValidateError(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectedErr error
	}{
		{
			"nil config",
			nil,
			ErrAuthServiceURLInvalid,
		},
		{
			"empty AuthServiceURL",
			&Config{
				AuthServiceURL: "",
			},
			ErrAuthServiceURLInvalid,
		},
		{
			"parse failed AuthServiceURL",
			&Config{
				AuthServiceURL: "http://[::1]:namedport",
			},
			ErrAuthServiceURLInvalid,
		},
		{
			"invalid scheme AuthServiceURL",
			&Config{
				AuthServiceURL: "ftp://auth.example.com",
			},
			ErrAuthServiceURLInvalid,
		},
		{
			"missing host AuthServiceURL",
			&Config{
				AuthServiceURL: "https://",
			},
			ErrAuthServiceURLInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if errors.Is(err, tt.expectedErr) == false {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
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
