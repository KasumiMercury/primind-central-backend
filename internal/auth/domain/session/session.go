package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserIDEmpty        = errors.New("user ID must be specified")
	ErrExpiresAtMissing   = errors.New("expiresAt must be specified")
	ErrExpiresBeforeStart = errors.New("expiresAt must be after createdAt")
	ErrSessionIDEmpty     = errors.New("session ID must be specified")
)

type ID string

func NewID() (ID, error) {
	return ID(uuid.NewString()), nil
}

func ParseID(id string) (ID, error) {
	candidate := ID(id)
	return candidate, candidate.validate()
}

func (id ID) String() string {
	return string(id)
}

func (id ID) Validate() error {
	return id.validate()
}

func (id ID) validate() error {
	if id == "" {
		return ErrSessionIDEmpty
	}
	if _, err := uuid.Parse(string(id)); err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}
	return nil
}

type Session struct {
	id        ID
	userID    string
	createdAt time.Time
	expiresAt time.Time
}

func NewSession(userID string, createdAt, expiresAt time.Time) (*Session, error) {
	id, err := NewID()
	if err != nil {
		return nil, err
	}
	return newSession(id, userID, createdAt, expiresAt)
}

func NewSessionWithID(id ID, userID string, createdAt, expiresAt time.Time) (*Session, error) {
	return newSession(id, userID, createdAt, expiresAt)
}

func newSession(id ID, userID string, createdAt, expiresAt time.Time) (*Session, error) {
	if err := id.validate(); err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, ErrUserIDEmpty
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	if expiresAt.IsZero() {
		return nil, ErrExpiresAtMissing
	}
	if !expiresAt.After(createdAt) {
		return nil, ErrExpiresBeforeStart
	}

	return &Session{
		id:        id,
		userID:    userID,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}, nil
}

func (s *Session) ID() ID {
	return s.id
}

func (s *Session) UserID() string {
	return s.userID
}

func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}
