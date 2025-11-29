package oidcidentity

import (
	"errors"
	"testing"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
)

func TestNewOIDCIdentitySuccess(t *testing.T) {
	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	tests := []struct {
		name     string
		userID   user.ID
		provider domainoidc.ProviderID
		subject  string
	}{
		{
			name:     "valid identity with google provider",
			userID:   userID,
			provider: domainoidc.ProviderGoogle,
			subject:  "google-user-123",
		},
		{
			name:     "valid identity with subject containing email",
			userID:   userID,
			provider: domainoidc.ProviderGoogle,
			subject:  "user@example.com",
		},
		{
			name:     "valid identity with numeric subject",
			userID:   userID,
			provider: domainoidc.ProviderGoogle,
			subject:  "1234567890",
		},
		{
			name:     "valid identity with complex subject",
			userID:   userID,
			provider: domainoidc.ProviderGoogle,
			subject:  "oauth2|google|abc123xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := NewOIDCIdentity(tt.userID, tt.provider, tt.subject)

			if err != nil {
				t.Fatalf("NewOIDCIdentity() unexpected error: %v", err)
			}

			if identity == nil {
				t.Fatal("NewOIDCIdentity() returned nil")
			}

			if identity.UserID().String() != tt.userID.String() {
				t.Errorf("identity.UserID() = %q, want %q", identity.UserID().String(), tt.userID.String())
			}

			if identity.Provider() != tt.provider {
				t.Errorf("identity.Provider() = %q, want %q", identity.Provider(), tt.provider)
			}

			if identity.Subject() != tt.subject {
				t.Errorf("identity.Subject() = %q, want %q", identity.Subject(), tt.subject)
			}
		})
	}
}

func TestNewOIDCIdentityErrors(t *testing.T) {
	validUserID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	emptyUserID := user.ID{}

	tests := []struct {
		name        string
		userID      user.ID
		provider    domainoidc.ProviderID
		subject     string
		expectedErr error
	}{
		{
			name:        "empty user ID",
			userID:      emptyUserID,
			provider:    domainoidc.ProviderGoogle,
			subject:     "valid-subject",
			expectedErr: ErrUserIDEmpty,
		},
		{
			name:        "empty provider",
			userID:      validUserID,
			provider:    "",
			subject:     "valid-subject",
			expectedErr: ErrProviderEmpty,
		},
		{
			name:        "empty subject",
			userID:      validUserID,
			provider:    domainoidc.ProviderGoogle,
			subject:     "",
			expectedErr: ErrSubjectEmpty,
		},
		{
			name:        "empty user ID and provider",
			userID:      emptyUserID,
			provider:    "",
			subject:     "valid-subject",
			expectedErr: ErrUserIDEmpty,
		},
		{
			name:        "empty user ID and subject",
			userID:      emptyUserID,
			provider:    domainoidc.ProviderGoogle,
			subject:     "",
			expectedErr: ErrUserIDEmpty,
		},
		{
			name:        "empty provider and subject",
			userID:      validUserID,
			provider:    "",
			subject:     "",
			expectedErr: ErrProviderEmpty,
		},
		{
			name:        "all fields empty",
			userID:      emptyUserID,
			provider:    "",
			subject:     "",
			expectedErr: ErrUserIDEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := NewOIDCIdentity(tt.userID, tt.provider, tt.subject)

			if err == nil {
				t.Fatalf("NewOIDCIdentity() expected error %v, got nil (identity: %+v)", tt.expectedErr, identity)
			}

			if identity != nil {
				t.Errorf("NewOIDCIdentity() expected nil identity on error, got: %+v", identity)
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("NewOIDCIdentity() error = %v, want %v", err, tt.expectedErr)
			}
		})
	}
}

func TestOIDCIdentityAccessors(t *testing.T) {
	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	const testProvider = domainoidc.ProviderGoogle
	const testSubject = "test-subject-123"

	identity, err := NewOIDCIdentity(userID, testProvider, testSubject)
	if err != nil {
		t.Fatalf("setup: NewOIDCIdentity() error: %v", err)
	}

	tests := []struct {
		name     string
		accessor func(*OIDCIdentity) interface{}
		expected interface{}
	}{
		{
			name: "UserID returns correct value",
			accessor: func(i *OIDCIdentity) interface{} {
				return i.UserID().String()
			},
			expected: userID.String(),
		},
		{
			name: "Provider returns correct value",
			accessor: func(i *OIDCIdentity) interface{} {
				return i.Provider()
			},
			expected: testProvider,
		},
		{
			name: "Subject returns correct value",
			accessor: func(i *OIDCIdentity) interface{} {
				return i.Subject()
			},
			expected: testSubject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.accessor(identity)
			if actual != tt.expected {
				t.Errorf("accessor returned %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestOIDCIdentityWithDifferentProviders(t *testing.T) {
	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	tests := []struct {
		name     string
		provider domainoidc.ProviderID
		subject  string
	}{
		{
			name:     "Google provider",
			provider: domainoidc.ProviderGoogle,
			subject:  "google-oauth-user-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := NewOIDCIdentity(userID, tt.provider, tt.subject)

			if err != nil {
				t.Fatalf("NewOIDCIdentity() error: %v", err)
			}

			if identity.Provider() != tt.provider {
				t.Errorf("identity.Provider() = %q, want %q", identity.Provider(), tt.provider)
			}

			if identity.Subject() != tt.subject {
				t.Errorf("identity.Subject() = %q, want %q", identity.Subject(), tt.subject)
			}
		})
	}
}
