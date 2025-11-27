package oidc

import (
	"errors"
	"testing"
	"time"
)

func TestNewParamsSuccess(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, time.January, 2, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name       string
		createdAt  time.Time
		expectAuto bool
	}{
		{
			name:      "explicit createdAt",
			createdAt: fixedTime,
		},
		{
			name:       "defaults createdAt to now",
			expectAuto: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var before time.Time
			if tt.expectAuto {
				before = time.Now().UTC()
			}

			params, err := NewParams(ProviderGoogle, "state-123", "nonce-abc", "verifier-xyz", tt.createdAt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if params.Provider() != ProviderGoogle {
				t.Fatalf("Provider() = %s, want %s", params.Provider(), ProviderGoogle)
			}
			if params.State() != "state-123" {
				t.Fatalf("State() = %s, want %s", params.State(), "state-123")
			}
			if params.Nonce() != "nonce-abc" {
				t.Fatalf("Nonce() = %s, want %s", params.Nonce(), "nonce-abc")
			}
			if params.CodeVerifier() != "verifier-xyz" {
				t.Fatalf("CodeVerifier() = %s, want %s", params.CodeVerifier(), "verifier-xyz")
			}

			if tt.expectAuto {
				after := time.Now().UTC()
				if params.CreatedAt().IsZero() {
					t.Fatalf("CreatedAt should be populated")
				}
				if params.CreatedAt().Before(before) || params.CreatedAt().After(after) {
					t.Fatalf("CreatedAt should be within call window [%s, %s], got %s", before, after, params.CreatedAt())
				}
			} else if !params.CreatedAt().Equal(tt.createdAt) {
				t.Fatalf("CreatedAt = %s, want %s", params.CreatedAt(), tt.createdAt)
			}
		})
	}
}

func TestNewParamsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		provider     ProviderID
		state        string
		nonce        string
		codeVerifier string
		createdAt    time.Time
		wantErrIs    error
	}{
		{
			name:         "missing provider",
			state:        "state-123",
			nonce:        "nonce-abc",
			codeVerifier: "verifier-xyz",
			wantErrIs:    ErrProviderInvalid,
		},
		{
			name:         "missing state",
			provider:     ProviderGoogle,
			nonce:        "nonce-abc",
			codeVerifier: "verifier-xyz",
			wantErrIs:    ErrStateEmpty,
		},
		{
			name:         "missing nonce",
			provider:     ProviderGoogle,
			state:        "state-123",
			codeVerifier: "verifier-xyz",
			wantErrIs:    ErrNonceEmpty,
		},
		{
			name:      "missing code verifier",
			provider:  ProviderGoogle,
			state:     "state-123",
			nonce:     "nonce-abc",
			wantErrIs: ErrCodeVerifierEmpty,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params, err := NewParams(tt.provider, tt.state, tt.nonce, tt.codeVerifier, tt.createdAt)
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("expected error %v, got %v", tt.wantErrIs, err)
			}
			if params != nil {
				t.Fatalf("expected params to be nil when error occurs")
			}
		})
	}
}

func TestParamsExpiresAt(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2025, time.January, 2, 15, 4, 5, 0, time.UTC)
	params, err := NewParams(ProviderGoogle, "state-123", "nonce-abc", "verifier-xyz", createdAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := createdAt.Add(ParamsExpirationDuration)
	if !params.ExpiresAt().Equal(want) {
		t.Errorf("ExpiresAt() = %s, want %s", params.ExpiresAt(), want)
	}
}

func TestParamsIsExpired(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2025, time.January, 2, 15, 0, 0, 0, time.UTC)
	params, err := NewParams(ProviderGoogle, "state-123", "nonce-abc", "verifier-xyz", createdAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name        string
		checkTime   time.Time
		wantExpired bool
	}{
		{
			name:        "before expiration",
			checkTime:   createdAt.Add(5 * time.Minute),
			wantExpired: false,
		},
		{
			name:        "exactly at expiration",
			checkTime:   createdAt.Add(ParamsExpirationDuration),
			wantExpired: false,
		},
		{
			name:        "after expiration",
			checkTime:   createdAt.Add(ParamsExpirationDuration + time.Millisecond),
			wantExpired: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := params.IsExpired(tt.checkTime); got != tt.wantExpired {
				t.Errorf("IsExpired(%s) = %v, want %v", tt.checkTime, got, tt.wantExpired)
			}
		})
	}
}
