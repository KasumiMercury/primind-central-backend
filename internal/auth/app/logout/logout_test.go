package logout

import (
	"context"
	"errors"
	"testing"
	"time"

	sessionCfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/jwt"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/testutil"
)

func TestLogoutSuccess(t *testing.T) {
	sessionRepo := setupSessionRepo(t)
	ctx := context.Background()

	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("failed to create user id: %v", err)
	}

	session, err := domainsession.NewSession(userID, timeNow(), timeNow().Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := sessionRepo.SaveSession(ctx, session); err != nil {
		t.Fatalf("failed to persist session: %v", err)
	}

	cfg := &sessionCfg.Config{Duration: time.Hour, Secret: "test-secret"}
	jwtGenerator := jwt.NewSessionJWTGenerator(cfg)
	jwtValidator := jwt.NewSessionJWTValidator(cfg)

	sessionToken, err := jwtGenerator.Generate(session, user.NewUser(userID, user.MustColor("#000000")))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	handler := NewLogoutHandler(sessionRepo, jwtValidator)

	resp, err := handler.Logout(context.Background(), &LogoutRequest{
		SessionToken: sessionToken,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected success response")
	}

	if userID == (user.ID{}) {
		t.Fatalf("user id should not be zero")
	}
}

func TestLogoutError(t *testing.T) {
	tests := []struct {
		name        string
		req         *LogoutRequest
		repo        domainsession.SessionRepository
		verifier    TokenVerifier
		expectedErr error
	}{
		{
			name:        "nil request",
			req:         nil,
			repo:        setupSessionRepo(t),
			verifier:    jwt.NewSessionJWTValidator(&sessionCfg.Config{Secret: "x", Duration: time.Hour}),
			expectedErr: errors.New("logout request is nil"),
		},
		{
			name:        "empty token",
			req:         &LogoutRequest{SessionToken: ""},
			repo:        setupSessionRepo(t),
			verifier:    jwt.NewSessionJWTValidator(&sessionCfg.Config{Secret: "x", Duration: time.Hour}),
			expectedErr: ErrTokenRequired,
		},
		{
			name: "verification failed",
			req: &LogoutRequest{
				SessionToken: "token",
			},
			repo:        setupSessionRepo(t),
			verifier:    jwt.NewSessionJWTValidator(&sessionCfg.Config{Secret: "wrong", Duration: time.Hour}),
			expectedErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := NewLogoutHandler(tt.repo, tt.verifier)

			_, err := handler.Logout(context.Background(), tt.req)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}

			if !errors.Is(err, tt.expectedErr) && err.Error() != tt.expectedErr.Error() {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func setupSessionRepo(t *testing.T) domainsession.SessionRepository {
	t.Helper()

	ctx := context.Background()
	redisClient, cleanupRedis := testutil.SetupRedisContainer(ctx, t)
	t.Cleanup(cleanupRedis)

	return repository.NewSessionRepository(redisClient)
}

func timeNow() time.Time {
	return time.Now().UTC()
}
