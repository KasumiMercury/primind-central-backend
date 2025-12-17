package config

import (
	"errors"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name                string
		envAuthServiceURL   string
		envDeviceServiceURL string
		envPrimindTasksURL  string
		expected            *Config
	}{
		{
			"valid URLs",
			"https://auth.example.com",
			"https://device.example.com",
			"https://tasks.example.com",
			&Config{
				AuthServiceURL:   "https://auth.example.com",
				DeviceServiceURL: "https://device.example.com",
			},
		},
		{
			"missing PRIMIND_TASKS_URL",
			"https://auth.example.com",
			"https://device.example.com",
			"",
			&Config{
				AuthServiceURL:   "https://auth.example.com",
				DeviceServiceURL: "https://device.example.com",
			},
		},
		{
			"default URLs",
			"",
			"",
			"http://localhost:8090",
			&Config{
				AuthServiceURL:   "http://localhost:8080",
				DeviceServiceURL: "http://localhost:8080",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AUTH_SERVICE_URL", tt.envAuthServiceURL)
			t.Setenv("DEVICE_SERVICE_URL", tt.envDeviceServiceURL)
			t.Setenv("PRIMIND_TASKS_URL", tt.envPrimindTasksURL)

			got, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}

			if got.AuthServiceURL != tt.expected.AuthServiceURL {
				t.Fatalf("Load() AuthServiceURL = %s, want %s", got.AuthServiceURL, tt.expected.AuthServiceURL)
			}

			if got.DeviceServiceURL != tt.expected.DeviceServiceURL {
				t.Fatalf("Load() DeviceServiceURL = %s, want %s", got.DeviceServiceURL, tt.expected.DeviceServiceURL)
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
				AuthServiceURL:   "https://auth.example.com",
				DeviceServiceURL: "https://device.example.com",
				TaskQueue: TaskQueueConfig{
					PrimindTasksURL: "https://tasks.example.com",
				},
			},
		},
		{
			"empty PRIMIND_TASKS_URL",
			Config{
				AuthServiceURL:   "https://auth.example.com",
				DeviceServiceURL: "https://device.example.com",
				TaskQueue: TaskQueueConfig{
					PrimindTasksURL: "",
				},
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
				AuthServiceURL:   "",
				DeviceServiceURL: "https://device.example.com",
			},
			ErrAuthServiceURLInvalid,
		},
		{
			"parse failed AuthServiceURL",
			&Config{
				AuthServiceURL:   "http://[::1]:namedport",
				DeviceServiceURL: "https://device.example.com",
			},
			ErrAuthServiceURLInvalid,
		},
		{
			"invalid scheme AuthServiceURL",
			&Config{
				AuthServiceURL:   "ftp://auth.example.com",
				DeviceServiceURL: "https://device.example.com",
			},
			ErrAuthServiceURLInvalid,
		},
		{
			"missing host AuthServiceURL",
			&Config{
				AuthServiceURL:   "https://",
				DeviceServiceURL: "https://device.example.com",
			},
			ErrAuthServiceURLInvalid,
		},
		{
			"empty DeviceServiceURL",
			&Config{
				AuthServiceURL:   "https://auth.example.com",
				DeviceServiceURL: "",
			},
			ErrDeviceServiceURLInvalid,
		},
		{
			"invalid scheme DeviceServiceURL",
			&Config{
				AuthServiceURL:   "https://auth.example.com",
				DeviceServiceURL: "ftp://device.example.com",
			},
			ErrDeviceServiceURLInvalid,
		},
		{
			"missing host DeviceServiceURL",
			&Config{
				AuthServiceURL:   "https://auth.example.com",
				DeviceServiceURL: "https://",
			},
			ErrDeviceServiceURLInvalid,
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
