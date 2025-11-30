package oidc

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGenerateCodeChallenge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		verifier      string
		wantChallenge string
	}{
		{
			name:     "generates correct SHA256 code challenge",
			verifier: "test-verifier-12345678901234567890123456",
			wantChallenge: func() string {
				hash := sha256.Sum256([]byte("test-verifier-12345678901234567890123456"))

				return base64.RawURLEncoding.EncodeToString(hash[:])
			}(),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			challenge := generateCodeChallenge(tt.verifier)

			if challenge != tt.wantChallenge {
				t.Errorf("generateCodeChallenge() = %v, want %v", challenge, tt.wantChallenge)
			}

			if len(challenge) != 43 {
				t.Errorf("challenge length = %d, want 43", len(challenge))
			}
		})
	}
}

func TestRandomToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "generates random token"},
		{name: "token is base64url encoded"},
		{name: "token is correct length"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := randomToken()
			if err != nil {
				t.Fatalf("randomToken() error: %v", err)
			}

			if token == "" {
				t.Error("randomToken() returned empty string")
			}

			if len(token) != 43 {
				t.Errorf("token length = %d, want 43", len(token))
			}

			_, err = base64.RawURLEncoding.DecodeString(token)
			if err != nil {
				t.Errorf("token is not valid base64url: %v", err)
			}
		})
	}
}

func TestRandomTokenUniqueness(t *testing.T) {
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		token, err := randomToken()
		if err != nil {
			t.Fatalf("randomToken() error: %v", err)
		}

		if seen[token] {
			t.Errorf("randomToken() generated duplicate token: %s", token)
		}

		seen[token] = true
	}
}
