package oidc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	oidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	sessionCfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity"
	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/clock"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/jwt"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/testutil"
	"go.uber.org/mock/gomock"
)

func TestLoginSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	provider := oidc.NewMockOIDCProviderWithLogin(ctrl)
	provider.EXPECT().
		ExchangeToken(gomock.Any(), "code-123", "code-verifier", "nonce-xyz").
		Return(&oidc.IDToken{
			Subject: "subject-123",
			Name:    "Jane Doe",
			Nonce:   "nonce-xyz",
		}, nil)

	params, _ := domainoidc.NewParams(domainoidc.ProviderGoogle, "state-1", "nonce-xyz", "code-verifier", now.Add(-time.Minute))

	repos := setupLoginReposWithClock(t, clock.NewFixedClock(now))
	t.Cleanup(repos.cleanup)

	if err := repos.paramsRepo.SaveParams(context.Background(), params); err != nil {
		t.Fatalf("failed to save params: %v", err)
	}

	handler := oidc.NewLoginHandlerWithClock(
		map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
			domainoidc.ProviderGoogle: provider,
		},
		repos.paramsRepo,
		repos.sessionRepo,
		repos.userRepo,
		repos.identityRepo,
		repos.userIdentityRepo,
		repos.jwtGenerator,
		repos.sessionCfg,
		clock.NewFixedClock(now),
	)

	result, err := handler.Login(context.Background(), &oidc.LoginRequest{
		Provider: domainoidc.ProviderGoogle,
		Code:     "code-123",
		State:    "state-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SessionToken == "" {
		t.Fatalf("expected signed token")
	}
}

func TestLoginError(t *testing.T) {
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	defaultParams, _ := domainoidc.NewParams(domainoidc.ProviderGoogle, "state-1", "nonce-xyz", "code-verifier", now.Add(-time.Minute))

	tests := []struct {
		name        string
		setup       func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase
		req         *oidc.LoginRequest
		expectedErr error
	}{
		{
			name: "unsupported provider",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				repos := setupLoginRepos(t)
				t.Cleanup(repos.cleanup)

				return oidc.NewLoginHandler(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{},
					repos.paramsRepo,
					repos.sessionRepo,
					repos.userRepo,
					repos.identityRepo,
					repos.userIdentityRepo,
					repos.jwtGenerator,
					repos.sessionCfg,
				)
			},
			req: &oidc.LoginRequest{
				Provider: "unknown",
				Code:     "code",
				State:    "state",
			},
			expectedErr: oidc.ErrOIDCProviderUnsupported,
		},
		{
			name: "params not found",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				repos := setupLoginRepos(t)
				t.Cleanup(repos.cleanup)

				provider := oidc.NewMockOIDCProviderWithLogin(ctrl)

				return oidc.NewLoginHandler(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
						domainoidc.ProviderGoogle: provider,
					},
					repos.paramsRepo,
					repos.sessionRepo,
					repos.userRepo,
					repos.identityRepo,
					repos.userIdentityRepo,
					repos.jwtGenerator,
					repos.sessionCfg,
				)
			},
			req: &oidc.LoginRequest{
				Provider: domainoidc.ProviderGoogle,
				Code:     "code",
				State:    "missing",
			},
			expectedErr: oidc.ErrStateInvalid,
		},
		{
			name: "params expired",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				mockParams := domainoidc.NewMockParamsRepository(ctrl)
				expiredParams, _ := domainoidc.NewParams(domainoidc.ProviderGoogle, "state-1", "nonce-xyz", "code-verifier", now.Add(-2*domainoidc.ParamsExpirationDuration))
				mockParams.EXPECT().GetParamsByState(gomock.Any(), "state-1").Return(expiredParams, nil)

				return oidc.NewLoginHandler(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
						domainoidc.ProviderGoogle: oidc.NewMockOIDCProviderWithLogin(ctrl),
					},
					mockParams,
					oidc.NewMockSessionRepository(ctrl),
					oidc.NewMockUserRepository(ctrl),
					oidc.NewMockOIDCIdentityRepository(ctrl),
					oidc.NewMockUserWithOIDCIdentityRepository(ctrl),
					oidc.NewMockSessionTokenGenerator(ctrl),
					&sessionCfg.Config{Duration: time.Hour},
				)
			},
			req: &oidc.LoginRequest{
				Provider: domainoidc.ProviderGoogle,
				Code:     "code",
				State:    "state-1",
			},
			expectedErr: domainoidc.ErrParamsExpired,
		},
		{
			name: "provider mismatch",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				repos := setupLoginReposWithClock(t, clock.NewFixedClock(now))
				t.Cleanup(repos.cleanup)

				mismatchedParams, _ := domainoidc.NewParams("another", "state-1", "nonce", "code", now)
				if err := repos.paramsRepo.SaveParams(context.Background(), mismatchedParams); err != nil {
					t.Fatalf("failed to save params: %v", err)
				}

				provider := oidc.NewMockOIDCProviderWithLogin(ctrl)

				return oidc.NewLoginHandlerWithClock(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
						domainoidc.ProviderGoogle: provider,
					},
					repos.paramsRepo,
					repos.sessionRepo,
					repos.userRepo,
					repos.identityRepo,
					repos.userIdentityRepo,
					repos.jwtGenerator,
					repos.sessionCfg,
					clock.NewFixedClock(now),
				)
			},
			req: &oidc.LoginRequest{
				Provider: domainoidc.ProviderGoogle,
				Code:     "code",
				State:    "state-1",
			},
			expectedErr: oidc.ErrStateInvalid,
		},
		{
			name: "token exchange failed",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				repos := setupLoginReposWithClock(t, clock.NewFixedClock(now))
				t.Cleanup(repos.cleanup)

				if err := repos.paramsRepo.SaveParams(context.Background(), defaultParams); err != nil {
					t.Fatalf("failed to save params: %v", err)
				}

				provider := oidc.NewMockOIDCProviderWithLogin(ctrl)
				provider.EXPECT().
					ExchangeToken(gomock.Any(), "bad-code", "code-verifier", "nonce-xyz").
					Return(nil, errors.New("exchange failed"))

				return oidc.NewLoginHandlerWithClock(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
						domainoidc.ProviderGoogle: provider,
					},
					repos.paramsRepo,
					repos.sessionRepo,
					repos.userRepo,
					repos.identityRepo,
					repos.userIdentityRepo,
					repos.jwtGenerator,
					repos.sessionCfg,
					clock.NewFixedClock(now),
				)
			},
			req: &oidc.LoginRequest{
				Provider: domainoidc.ProviderGoogle,
				Code:     "bad-code",
				State:    "state-1",
			},
			expectedErr: oidc.ErrCodeInvalid,
		},
		{
			name: "nonce mismatch",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				repos := setupLoginReposWithClock(t, clock.NewFixedClock(now))
				t.Cleanup(repos.cleanup)

				if err := repos.paramsRepo.SaveParams(context.Background(), defaultParams); err != nil {
					t.Fatalf("failed to save params: %v", err)
				}

				provider := oidc.NewMockOIDCProviderWithLogin(ctrl)
				provider.EXPECT().
					ExchangeToken(gomock.Any(), "code-123", "code-verifier", "nonce-xyz").
					Return(&oidc.IDToken{
						Subject: "sub",
						Nonce:   "other",
					}, nil)

				return oidc.NewLoginHandlerWithClock(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
						domainoidc.ProviderGoogle: provider,
					},
					repos.paramsRepo,
					repos.sessionRepo,
					repos.userRepo,
					repos.identityRepo,
					repos.userIdentityRepo,
					repos.jwtGenerator,
					repos.sessionCfg,
					clock.NewFixedClock(now),
				)
			},
			req: &oidc.LoginRequest{
				Provider: domainoidc.ProviderGoogle,
				Code:     "code-123",
				State:    "state-1",
			},
			expectedErr: oidc.ErrNonceInvalid,
		},
		{
			name: "identity lookup error",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				repos := setupLoginReposWithClock(t, clock.NewFixedClock(now))
				t.Cleanup(repos.cleanup)

				if err := repos.paramsRepo.SaveParams(context.Background(), defaultParams); err != nil {
					t.Fatalf("failed to save params: %v", err)
				}

				provider := oidc.NewMockOIDCProviderWithLogin(ctrl)
				provider.EXPECT().
					ExchangeToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&oidc.IDToken{
						Subject: "sub",
						Nonce:   "nonce-xyz",
					}, nil)

				mockIdentity := oidc.NewMockOIDCIdentityRepository(ctrl)
				mockIdentity.EXPECT().
					GetOIDCIdentityByProviderSubject(gomock.Any(), domainoidc.ProviderGoogle, "sub").
					Return(nil, errors.New("db down"))

				return oidc.NewLoginHandlerWithClock(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
						domainoidc.ProviderGoogle: provider,
					},
					repos.paramsRepo,
					repos.sessionRepo,
					repos.userRepo,
					mockIdentity,
					repos.userIdentityRepo,
					repos.jwtGenerator,
					repos.sessionCfg,
					clock.NewFixedClock(now),
				)
			},
			req: &oidc.LoginRequest{
				Provider: domainoidc.ProviderGoogle,
				Code:     "code-123",
				State:    "state-1",
			},
			expectedErr: errors.New("db down"),
		},
		{
			name: "existing user load failure",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				repos := setupLoginReposWithClock(t, clock.NewFixedClock(now))
				t.Cleanup(repos.cleanup)

				if err := repos.paramsRepo.SaveParams(context.Background(), defaultParams); err != nil {
					t.Fatalf("failed to save params: %v", err)
				}

				provider := oidc.NewMockOIDCProviderWithLogin(ctrl)
				provider.EXPECT().
					ExchangeToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&oidc.IDToken{
						Subject: "subject-123",
						Nonce:   "nonce-xyz",
					}, nil)

				userID, _ := user.NewID()
				identity, _ := oidcidentity.NewOIDCIdentity(userID, domainoidc.ProviderGoogle, "subject-123")

				mockUserRepo := oidc.NewMockUserRepository(ctrl)
				mockUserRepo.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(nil, errors.New("load failed"))

				mockIdentity := oidc.NewMockOIDCIdentityRepository(ctrl)
				mockIdentity.EXPECT().
					GetOIDCIdentityByProviderSubject(gomock.Any(), domainoidc.ProviderGoogle, "subject-123").
					Return(identity, nil)

				return oidc.NewLoginHandlerWithClock(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
						domainoidc.ProviderGoogle: provider,
					},
					repos.paramsRepo,
					repos.sessionRepo,
					mockUserRepo,
					mockIdentity,
					repos.userIdentityRepo,
					repos.jwtGenerator,
					repos.sessionCfg,
					clock.NewFixedClock(now),
				)
			},
			req: &oidc.LoginRequest{
				Provider: domainoidc.ProviderGoogle,
				Code:     "code-123",
				State:    "state-1",
			},
			expectedErr: errors.New("load failed"),
		},
		{
			name: "session save failure",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				repos := setupLoginReposWithClock(t, clock.NewFixedClock(now))
				t.Cleanup(repos.cleanup)

				if err := repos.paramsRepo.SaveParams(context.Background(), defaultParams); err != nil {
					t.Fatalf("failed to save params: %v", err)
				}

				provider := oidc.NewMockOIDCProviderWithLogin(ctrl)
				provider.EXPECT().
					ExchangeToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&oidc.IDToken{
						Subject: "new-subject",
						Nonce:   "nonce-xyz",
					}, nil)

				mockSessionRepo := oidc.NewMockSessionRepository(ctrl)
				mockSessionRepo.EXPECT().
					SaveSession(gomock.Any(), gomock.Any()).
					Return(errors.New("save failed"))

				return oidc.NewLoginHandlerWithClock(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
						domainoidc.ProviderGoogle: provider,
					},
					repos.paramsRepo,
					mockSessionRepo,
					repos.userRepo,
					repos.identityRepo,
					repos.userIdentityRepo,
					repos.jwtGenerator,
					repos.sessionCfg,
					clock.NewFixedClock(now),
				)
			},
			req: &oidc.LoginRequest{
				Provider: domainoidc.ProviderGoogle,
				Code:     "code-123",
				State:    "state-1",
			},
			expectedErr: errors.New("save failed"),
		},
		{
			name: "jwt generation failure",
			setup: func(ctrl *gomock.Controller) oidc.OIDCLoginUseCase {
				repos := setupLoginReposWithClock(t, clock.NewFixedClock(now))
				t.Cleanup(repos.cleanup)

				if err := repos.paramsRepo.SaveParams(context.Background(), defaultParams); err != nil {
					t.Fatalf("failed to save params: %v", err)
				}

				provider := oidc.NewMockOIDCProviderWithLogin(ctrl)
				provider.EXPECT().
					ExchangeToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&oidc.IDToken{
						Subject: "new-subject",
						Nonce:   "nonce-xyz",
					}, nil)

				mockJWT := oidc.NewMockSessionTokenGenerator(ctrl)
				mockJWT.EXPECT().
					Generate(gomock.Any(), gomock.Any()).
					Return("", errors.New("sign failed"))

				return oidc.NewLoginHandlerWithClock(
					map[domainoidc.ProviderID]oidc.OIDCProviderWithLogin{
						domainoidc.ProviderGoogle: provider,
					},
					repos.paramsRepo,
					repos.sessionRepo,
					repos.userRepo,
					repos.identityRepo,
					repos.userIdentityRepo,
					mockJWT,
					repos.sessionCfg,
					clock.NewFixedClock(now),
				)
			},
			req: &oidc.LoginRequest{
				Provider: domainoidc.ProviderGoogle,
				Code:     "code-123",
				State:    "state-1",
			},
			expectedErr: errors.New("sign failed"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler := tt.setup(ctrl)

			_, err := handler.Login(context.Background(), tt.req)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}

			if !errors.Is(err, tt.expectedErr) && err.Error() != tt.expectedErr.Error() {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

type loginRepos struct {
	paramsRepo       domainoidc.ParamsRepository
	sessionRepo      domainsession.SessionRepository
	userRepo         user.UserRepository
	identityRepo     oidcidentity.OIDCIdentityRepository
	userIdentityRepo oidc.UserWithOIDCIdentityRepository
	jwtGenerator     oidc.SessionTokenGenerator
	sessionCfg       *sessionCfg.Config
	cleanup          func()
}

func setupLoginReposWithClock(t *testing.T, clk clock.Clock) loginRepos {
	t.Helper()

	ctx := context.Background()

	redisClient, cleanupRedis := testutil.SetupRedisContainer(ctx, t)
	postgresDB, cleanupPostgres := testutil.SetupPostgresContainer(ctx, t)

	if redisClient == nil {
		t.Skip("redis container unavailable")
	}

	if postgresDB == nil {
		t.Skip("postgres container unavailable")
	}

	if err := postgresDB.AutoMigrate(&repository.UserModel{}, &repository.OIDCIdentityModel{}); err != nil {
		t.Skipf("failed to migrate tables: %v", err)
	}

	cleanup := func() {
		cleanupRedis()
		cleanupPostgres()
	}

	cfg := &sessionCfg.Config{Duration: time.Hour, Secret: "integration-secret"}

	var (
		paramsRepo       domainoidc.ParamsRepository
		sessionRepo      domainsession.SessionRepository
		userRepo         user.UserRepository
		identityRepo     oidcidentity.OIDCIdentityRepository
		userIdentityRepo oidc.UserWithOIDCIdentityRepository
	)

	if clk != nil {
		paramsRepo = repository.NewOIDCParamsRepositoryWithClock(redisClient, clk)
		sessionRepo = repository.NewSessionRepositoryWithClock(redisClient, clk)
		userRepo = repository.NewUserRepositoryWithClock(postgresDB, clk)
		identityRepo = repository.NewOIDCIdentityRepositoryWithClock(postgresDB, clk)
		userIdentityRepo = repository.NewUserWithIdentityRepositoryWithClock(postgresDB, clk)
	} else {
		paramsRepo = repository.NewOIDCParamsRepository(redisClient)
		sessionRepo = repository.NewSessionRepository(redisClient)
		userRepo = repository.NewUserRepository(postgresDB)
		identityRepo = repository.NewOIDCIdentityRepository(postgresDB)
		userIdentityRepo = repository.NewUserWithIdentityRepository(postgresDB)
	}

	return loginRepos{
		paramsRepo:       paramsRepo,
		sessionRepo:      sessionRepo,
		userRepo:         userRepo,
		identityRepo:     identityRepo,
		userIdentityRepo: userIdentityRepo,
		jwtGenerator:     jwt.NewSessionJWTGenerator(cfg),
		sessionCfg:       cfg,
		cleanup:          cleanup,
	}
}

func setupLoginRepos(t *testing.T) loginRepos {
	return setupLoginReposWithClock(t, nil)
}
