package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/google/uuid"
)

func mustUserID(t *testing.T) user.ID {
	t.Helper()
	id, err := user.NewID()
	if err != nil {
		t.Fatalf("failed to create user ID: %v", err)
	}
	return id
}

func TestParseIDSuccess(t *testing.T) {
	t.Parallel()

	validID, err := NewID()
	if err != nil {
		t.Fatalf("NewID returned error: %v", err)
	}

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "generated uuid",
			input: validID.String(),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := ParseID(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := id.Validate(); err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
		})
	}
}

func TestParseIDErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantErrIs error
	}{
		{
			name:      "empty id",
			input:     "",
			wantErrIs: ErrSessionIDEmpty,
		},
		{
			name:      "invalid uuid",
			input:     "not-a-uuid",
			wantErrIs: ErrSessionIDInvalidFormat,
		},
		{
			name:      "uuid but not v7",
			input:     "550e8400-e29b-41d4-a716-446655440000",
			wantErrIs: ErrSessionIDInvalidV7,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := ParseID(tt.input)
			if err == nil {
				t.Fatalf("expected error but got nil (id: %v)", id)
			}

			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("expected error %v, got %v", tt.wantErrIs, err)
			}
		})
	}
}

func TestNewSessionSuccess(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, time.January, 2, 15, 4, 5, 0, time.UTC)
	expires := baseTime.Add(2 * time.Hour)

	customID, err := NewID()
	if err != nil {
		t.Fatalf("NewID returned error: %v", err)
	}

	testUserID := mustUserID(t)

	tests := []struct {
		name       string
		build      func() (*Session, time.Time, time.Time, error)
		expectAuto bool
		wantID     *ID
		wantUser   user.ID
	}{
		{
			name: "provided times",
			build: func() (*Session, time.Time, time.Time, error) {
				s, err := NewSession(testUserID, baseTime, expires)

				return s, baseTime, expires, err
			},
			wantUser: testUserID,
		},
		{
			name: "createdAt defaults to now",
			build: func() (*Session, time.Time, time.Time, error) {
				exp := time.Now().UTC().Add(30 * time.Minute)
				s, err := NewSession(testUserID, time.Time{}, exp)

				return s, time.Time{}, exp, err
			},
			expectAuto: true,
			wantUser:   testUserID,
		},
		{
			name: "custom session id",
			build: func() (*Session, time.Time, time.Time, error) {
				s, err := NewSessionWithID(customID, testUserID, baseTime, expires)

				return s, baseTime, expires, err
			},
			wantUser: testUserID,
			wantID:   &customID,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := time.Now().UTC()
			session, wantCreatedAt, wantExpiresAt, err := tt.build()
			after := time.Now().UTC()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if session == nil {
				t.Fatalf("session should not be nil")
			}

			if session.ID().Validate() != nil {
				t.Fatalf("session ID should be valid, got error: %v", session.ID().Validate())
			}

			if tt.wantID != nil && session.ID() != *tt.wantID {
				t.Fatalf("ID() = %s, want %s", session.ID().String(), tt.wantID.String())
			}

			if session.UserID() != tt.wantUser {
				t.Fatalf("UserID() = %s, want %s", session.UserID(), tt.wantUser)
			}

			if tt.expectAuto {
				if session.CreatedAt().Before(before) || session.CreatedAt().After(after) {
					t.Fatalf("CreatedAt should be within call window [%s, %s], got %s", before, after, session.CreatedAt())
				}
			} else if !session.CreatedAt().Equal(wantCreatedAt) {
				t.Fatalf("CreatedAt = %s, want %s", session.CreatedAt(), wantCreatedAt)
			}

			if session.ExpiresAt() != wantExpiresAt {
				t.Fatalf("ExpiresAt = %s, want %s", session.ExpiresAt(), wantExpiresAt)
			}
		})
	}
}

func TestNewSessionErrors(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, time.January, 2, 15, 4, 5, 0, time.UTC)
	expires := baseTime.Add(2 * time.Hour)
	testUserID := mustUserID(t)

	tests := []struct {
		name      string
		build     func() (*Session, error)
		wantErrIs error
	}{
		{
			name: "missing user id",
			build: func() (*Session, error) {
				return NewSession(user.ID{}, baseTime, expires)
			},
			wantErrIs: ErrUserIDEmpty,
		},
		{
			name: "missing expiresAt",
			build: func() (*Session, error) {
				return NewSession(testUserID, baseTime, time.Time{})
			},
			wantErrIs: ErrExpiresAtMissing,
		},
		{
			name: "expires before created",
			build: func() (*Session, error) {
				return NewSession(testUserID, baseTime, baseTime.Add(-time.Minute))
			},
			wantErrIs: ErrExpiresBeforeStart,
		},
		{
			name: "invalid custom session id",
			build: func() (*Session, error) {
				return NewSessionWithID(ID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")), testUserID, baseTime, expires)
			},
			wantErrIs: ErrSessionIDInvalidV7,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session, err := tt.build()
			if err == nil {
				t.Fatalf("expected error but got nil")
			}

			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("expected error %v, got %v", tt.wantErrIs, err)
			}

			if session != nil {
				t.Fatalf("expected session to be nil when error occurs")
			}
		})
	}
}
