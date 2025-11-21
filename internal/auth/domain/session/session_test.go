package domain

import (
	"errors"
	"testing"
	"time"
)

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
			name:  "invalid uuid",
			input: "not-a-uuid",
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

	tests := []struct {
		name       string
		build      func() (*Session, time.Time, time.Time, error)
		expectAuto bool
		wantID     *ID
		wantUser   string
	}{
		{
			name: "provided times",
			build: func() (*Session, time.Time, time.Time, error) {
				s, err := NewSession("user-123", baseTime, expires)
				return s, baseTime, expires, err
			},
			wantUser: "user-123",
		},
		{
			name: "createdAt defaults to now",
			build: func() (*Session, time.Time, time.Time, error) {
				exp := time.Now().UTC().Add(30 * time.Minute)
				s, err := NewSession("user-123", time.Time{}, exp)
				return s, time.Time{}, exp, err
			},
			expectAuto: true,
			wantUser:   "user-123",
		},
		{
			name: "custom session id",
			build: func() (*Session, time.Time, time.Time, error) {
				s, err := NewSessionWithID(customID, "user-123", baseTime, expires)
				return s, baseTime, expires, err
			},
			wantUser: "user-123",
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
				t.Fatalf("ID() = %s, want %s", session.ID(), *tt.wantID)
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

	tests := []struct {
		name      string
		build     func() (*Session, error)
		wantErrIs error
	}{
		{
			name: "missing user id",
			build: func() (*Session, error) {
				return NewSession("", baseTime, expires)
			},
			wantErrIs: ErrUserIDEmpty,
		},
		{
			name: "missing expiresAt",
			build: func() (*Session, error) {
				return NewSession("user-123", baseTime, time.Time{})
			},
			wantErrIs: ErrExpiresAtMissing,
		},
		{
			name: "expires before created",
			build: func() (*Session, error) {
				return NewSession("user-123", baseTime, baseTime.Add(-time.Minute))
			},
			wantErrIs: ErrExpiresBeforeStart,
		},
		{
			name: "invalid custom session id",
			build: func() (*Session, error) {
				return NewSessionWithID(ID("invalid"), "user-123", baseTime, expires)
			},
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
