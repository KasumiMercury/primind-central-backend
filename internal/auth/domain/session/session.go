package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/google/uuid"
)

var (
	ErrUserIDEmpty            = errors.New("user ID must be specified")
	ErrExpiresAtMissing       = errors.New("expiresAt must be specified")
	ErrExpiresBeforeStart     = errors.New("expiresAt must be after createdAt")
	ErrSessionIDEmpty         = errors.New("session ID must be specified")
	ErrSessionIDInvalidFormat = errors.New("session ID must be a valid UUID")
	ErrSessionIDInvalidV7     = errors.New("session ID must be a UUIDv7")
	ErrSessionIDGeneration    = errors.New("failed to generate session ID")
)

type ID uuid.UUID

func NewID() (ID, error) {
	v7, err := uuid.NewV7()
	if err != nil {
		return ID{}, fmt.Errorf("%w: %v", ErrSessionIDGeneration, err)
	}

	return ID(v7), nil
}

func ParseID(id string) (ID, error) {
	if id == "" {
		return ID{}, ErrSessionIDEmpty
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		return ID{}, fmt.Errorf("%w: %v", ErrSessionIDInvalidFormat, err)
	}

	candidate := ID(parsed)

	return candidate, candidate.validate()
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

func (id ID) Validate() error {
	return id.validate()
}

func (id ID) validate() error {
	if uuid.UUID(id) == uuid.Nil {
		return ErrSessionIDEmpty
	}

	if uuid.UUID(id).Version() != 7 {
		return ErrSessionIDInvalidV7
	}

	return nil
}

type Session struct {
	id        ID
	userID    user.ID
	createdAt time.Time
	expiresAt time.Time
}

func NewSession(userID user.ID, createdAt, expiresAt time.Time) (*Session, error) {
	id, err := NewID()
	if err != nil {
		return nil, err
	}

	return newSession(id, userID, createdAt, expiresAt)
}

func NewSessionWithID(id ID, userID user.ID, createdAt, expiresAt time.Time) (*Session, error) {
	return newSession(id, userID, createdAt, expiresAt)
}

func newSession(id ID, userID user.ID, createdAt, expiresAt time.Time) (*Session, error) {
	if err := id.validate(); err != nil {
		return nil, err
	}

	if userID == (user.ID{}) {
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

func (s *Session) UserID() user.ID {
	return s.userID
}

func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}
