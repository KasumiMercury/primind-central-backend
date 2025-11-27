package session

import (
	"errors"
	"testing"
	"time"
)

func TestLoadSessionConfigSuccess(t *testing.T) {
	tests := []struct {
		name         string
		secret       string
		durationEnv  string
		wantDuration time.Duration
	}{
		{
			name:         "with custom duration",
			secret:       "super-secret",
			durationEnv:  "2h30m",
			wantDuration: 150 * time.Minute,
		},
		{
			name:         "default duration when env missing",
			secret:       "super-secret",
			durationEnv:  "",
			wantDuration: defaultSessionDuration,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(sessionSecretEnv, tt.secret)
			if tt.durationEnv != "" {
				t.Setenv(sessionDurationEnv, tt.durationEnv)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}

			if cfg.Secret != tt.secret {
				t.Fatalf("Secret = %s, want %s", cfg.Secret, tt.secret)
			}
			if cfg.Duration != tt.wantDuration {
				t.Fatalf("Duration = %s, want %s", cfg.Duration, tt.wantDuration)
			}
		})
	}
}

func TestLoadSessionConfigErrors(t *testing.T) {
	t.Run("missing secret", func(t *testing.T) {
		t.Setenv(sessionSecretEnv, "")
		_, err := Load()
		if err == nil {
			t.Fatalf("expected error but got nil")
		}
		if !errors.Is(err, ErrSessionSecretMissing) {
			t.Fatalf("expected ErrSessionSecretMissing, got %v", err)
		}
	})
}

func TestSessionConfigValidateSuccess(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Secret:   "super-secret",
		Duration: 30 * time.Minute,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestSessionConfigValidateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *Config
		wantErr error
	}{
		{
			name: "missing secret",
			cfg: &Config{
				Secret:   "",
				Duration: time.Hour,
			},
			wantErr: ErrSessionSecretMissing,
		},
		{
			name: "non-positive duration",
			cfg: &Config{
				Secret:   "secret",
				Duration: -time.Minute,
			},
			wantErr: ErrSessionDurationInvalid,
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
