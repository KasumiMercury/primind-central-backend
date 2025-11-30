package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/testutil"
)

func TestSessionRepositoryIntegrationSuccess(t *testing.T) {
	ctx := context.Background()

	client, cleanup := testutil.SetupRedisContainer(ctx, t)
	defer cleanup()

	repo := NewSessionRepository(client)

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to create user id: %v", err)
	}

	now := time.Now().UTC()

	session, err := domainsession.NewSession(userID, now, now.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := repo.SaveSession(ctx, session); err != nil {
		t.Fatalf("SaveSession returned error: %v", err)
	}

	found, err := repo.GetSession(ctx, session.ID())
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}

	if found.ID() != session.ID() || found.UserID() != userID {
		t.Fatalf("unexpected session data")
	}

	if err := repo.DeleteSession(ctx, session.ID()); err != nil {
		t.Fatalf("DeleteSession returned error: %v", err)
	}
}

func TestSessionRepositoryIntegrationError(t *testing.T) {
	ctx := context.Background()

	client, cleanup := testutil.SetupRedisContainer(ctx, t)
	defer cleanup()

	repo := NewSessionRepository(client)

	if err := repo.SaveSession(ctx, nil); !errors.Is(err, ErrSessionRequired) {
		t.Fatalf("expected ErrSessionRequired, got %v", err)
	}

	userID, _ := domainuser.NewID()
	now := time.Now().UTC()

	expiredSession, _ := domainsession.NewSession(userID, now.Add(-time.Hour), now.Add(-30*time.Minute))
	if err := repo.SaveSession(ctx, expiredSession); !errors.Is(err, ErrSessionAlreadyExpired) {
		t.Fatalf("expected ErrSessionAlreadyExpired, got %v", err)
	}

	randomID, _ := domainsession.NewID()
	if _, err := repo.GetSession(ctx, randomID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
