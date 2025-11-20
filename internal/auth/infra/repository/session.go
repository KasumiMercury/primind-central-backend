package repository

import (
	"context"
	"errors"
	"sync"

	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
)

var ErrSessionNotFound = errors.New("session not found")

type inMemorySessionRepository struct {
	mu          sync.Mutex
	bySessionID map[string]domain.Session
}

func NewInMemorySessionRepository() domain.SessionRepository {
	return &inMemorySessionRepository{
		bySessionID: make(map[string]domain.Session),
	}
}

func (r *inMemorySessionRepository) SaveSession(_ context.Context, session *domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.bySessionID[session.ID] = *session
	return nil
}

func (r *inMemorySessionRepository) GetSession(_ context.Context, sessionID string) (*domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.bySessionID[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return &session, nil
}
