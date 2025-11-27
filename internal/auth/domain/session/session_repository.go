package domain

import "context"

type SessionRepository interface {
	SaveSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, sessionID ID) (*Session, error)
}
