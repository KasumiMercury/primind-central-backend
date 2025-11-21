package domain

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
)

var (
	ErrUserIDEmpty        = errors.New("user ID must be specified")
	ErrExpiresAtMissing   = errors.New("expiresAt must be specified")
	ErrExpiresBeforeStart = errors.New("expiresAt must be after createdAt")
	ErrSessionIDEmpty     = errors.New("session ID must be specified")
)

type Session struct {
	id        string
	userID    string
	createdAt time.Time
	expiresAt time.Time
}

func NewSession(userID string, createdAt, expiresAt time.Time) (*Session, error) {
	id, err := randomToken()
	if err != nil {
		return nil, err
	}
	return newSession(id, userID, createdAt, expiresAt)
}

func NewSessionWithID(id, userID string, createdAt, expiresAt time.Time) (*Session, error) {
	return newSession(id, userID, createdAt, expiresAt)
}

func newSession(id, userID string, createdAt, expiresAt time.Time) (*Session, error) {
	if id == "" {
		return nil, ErrSessionIDEmpty
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

func (s *Session) ID() string {
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

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
