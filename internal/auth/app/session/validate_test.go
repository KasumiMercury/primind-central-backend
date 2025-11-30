package session

import (
	"context"
	"errors"
	"testing"
	"time"

	sessionCfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/clock"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/jwt"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/testutil"
	"go.uber.org/mock/gomock"
)

func TestValidateSessionSuccess(t *testing.T) {
	now := time.Now().UTC()

	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("failed to create user id: %v", err)
	}

	session, err := domainsession.NewSession(userID, now, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	repo := setupSessionRepo(t)
	if err := repo.SaveSession(context.Background(), session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	cfg := &sessionCfg.Config{Duration: time.Hour, Secret: "test-secret"}
	jwtGen := jwt.NewSessionJWTGenerator(cfg)
	jwtValidator := jwt.NewSessionJWTValidator(cfg)

	token, err := jwtGen.Generate(session, user.NewUser(userID, user.MustColor("#000000")))
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	handler := newValidateSessionHandler(repo, jwtValidator, clock.NewFixedClock(now))

	result, err := handler.Validate(context.Background(), &ValidateSessionRequest{
		SessionToken: token,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID.String(), result.UserID.String())
	}
}

func TestValidateSessionError(t *testing.T) {
	now := time.Now().UTC()
	userID, _ := user.NewID()
	validSession, _ := domainsession.NewSession(userID, now.Add(-2*time.Hour), now.Add(-time.Hour))

	tests := []struct {
		name        string
		req         *ValidateSessionRequest
		repo        domainsession.SessionRepository
		verifier    TokenVerifier
		expectedErr error
	}{
		{
			name:        "nil request",
			req:         nil,
			repo:        setupSessionRepo(t),
			verifier:    jwt.NewSessionJWTValidator(&sessionCfg.Config{Secret: "x", Duration: time.Hour}),
			expectedErr: ErrRequestNil,
		},
		{
			name:        "empty token",
			req:         &ValidateSessionRequest{SessionToken: ""},
			repo:        setupSessionRepo(t),
			verifier:    jwt.NewSessionJWTValidator(&sessionCfg.Config{Secret: "x", Duration: time.Hour}),
			expectedErr: ErrSessionTokenRequired,
		},
		{
			name: "verification failed",
			req: &ValidateSessionRequest{
				SessionToken: "token",
			},
			repo:        setupSessionRepo(t),
			verifier:    jwt.NewSessionJWTValidator(&sessionCfg.Config{Secret: "wrong", Duration: time.Hour}),
			expectedErr: ErrSessionTokenInvalid,
		},
		{
			name: "extract session id failed",
			req: &ValidateSessionRequest{
				SessionToken: "token",
			},
			repo:        setupSessionRepo(t),
			verifier:    jwt.NewSessionJWTValidator(&sessionCfg.Config{Secret: "wrong", Duration: time.Hour}),
			expectedErr: ErrSessionTokenInvalid,
		},
		{
			name: "session missing",
			req: func() *ValidateSessionRequest {
				cfg := &sessionCfg.Config{Secret: "x", Duration: time.Hour}
				generator := jwt.NewSessionJWTGenerator(cfg)

				session, _ := domainsession.NewSession(userID, now, now.Add(time.Hour))
				token, _ := generator.Generate(session, user.NewUser(userID, user.MustColor("#000000")))

				return &ValidateSessionRequest{SessionToken: token}
			}(),
			repo:        setupSessionRepo(t),
			verifier:    jwt.NewSessionJWTValidator(&sessionCfg.Config{Secret: "x", Duration: time.Hour}),
			expectedErr: ErrSessionNotFound,
		},
		{
			name: "session expired",
			req: &ValidateSessionRequest{
				SessionToken: "expired-token",
			},
			repo: func() domainsession.SessionRepository {
				ctrl := gomock.NewController(t)
				t.Cleanup(ctrl.Finish)

				mockRepo := NewMockSessionRepository(ctrl)
				mockRepo.EXPECT().
					GetSession(gomock.Any(), validSession.ID()).
					Return(validSession, nil)

				return mockRepo
			}(),
			verifier: func() TokenVerifier {
				ctrl := gomock.NewController(t)
				t.Cleanup(ctrl.Finish)

				mockVerifier := NewMockTokenVerifier(ctrl)
				mockVerifier.EXPECT().Verify("expired-token").Return(nil)
				mockVerifier.EXPECT().ExtractSessionID("expired-token").Return(validSession.ID().String(), nil)

				return mockVerifier
			}(),
			expectedErr: ErrSessionExpired,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newValidateSessionHandler(tt.repo, tt.verifier, clock.NewFixedClock(now))

			_, err := handler.Validate(context.Background(), tt.req)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}

			if !errors.Is(err, tt.expectedErr) {
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
