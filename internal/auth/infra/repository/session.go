package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/redis/go-redis/v9"
)

var (
	ErrSessionRequired       = errors.New("session is required")
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionAlreadyExpired = errors.New("session already expired")
)

type sessionRecord struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type sessionRepository struct {
	client *redis.Client
	now    func() time.Time
}

func NewSessionRepository(client *redis.Client) domainsession.SessionRepository {
	return &sessionRepository{
		client: client,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (r *sessionRepository) SaveSession(ctx context.Context, session *domainsession.Session) error {
	if session == nil {
		return ErrSessionRequired
	}

	record := sessionRecord{
		ID:        session.ID().String(),
		UserID:    session.UserID().String(),
		CreatedAt: session.CreatedAt(),
		ExpiresAt: session.ExpiresAt(),
	}

	ttl := time.Until(session.ExpiresAt())
	if ttl <= 0 {
		return ErrSessionAlreadyExpired
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, r.key(session.ID().String()), payload, ttl).Err()
}

func (r *sessionRepository) GetSession(ctx context.Context, sessionID domainsession.ID) (*domainsession.Session, error) {
	raw, err := r.client.Get(ctx, r.key(sessionID.String())).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrSessionNotFound
	}

	if err != nil {
		return nil, err
	}

	var record sessionRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}

	uid, err := domainuser.NewIDFromString(record.UserID)
	if err != nil {
		return nil, err
	}

	parsedID, err := domainsession.ParseID(record.ID)
	if err != nil {
		return nil, err
	}

	return domainsession.NewSessionWithID(parsedID, uid, record.CreatedAt, record.ExpiresAt)
}

func (r *sessionRepository) DeleteSession(ctx context.Context, sessionID domainsession.ID) error {
	return r.client.Del(ctx, r.key(sessionID.String())).Err()
}

func (r *sessionRepository) key(sessionID string) string {
	return fmt.Sprintf("auth:session:%s", sessionID)
}
