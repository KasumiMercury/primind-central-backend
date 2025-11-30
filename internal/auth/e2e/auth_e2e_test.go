package e2e

import (
	"context"
	"errors"
	"testing"
	"time"

	applogout "github.com/KasumiMercury/primind-central-backend/internal/auth/app/logout"
	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	appsession "github.com/KasumiMercury/primind-central-backend/internal/auth/app/session"
	sessioncfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/session"
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/jwt"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	authsvc "github.com/KasumiMercury/primind-central-backend/internal/auth/infra/service"
	authv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/auth/v1"
	"github.com/KasumiMercury/primind-central-backend/internal/testutil"
	"go.uber.org/mock/gomock"
)

func TestAuthE2ELoginValidateLogoutFlow(t *testing.T) {
	ctx := context.Background()

	redisClient, cleanupRedis := testutil.SetupRedisContainer(ctx, t)
	defer cleanupRedis()

	db, cleanupPostgres := testutil.SetupPostgresContainer(ctx, t)
	defer cleanupPostgres()

	if err := db.AutoMigrate(&repository.UserModel{}, &repository.OIDCIdentityModel{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	paramsRepo := repository.NewOIDCParamsRepository(redisClient)
	sessionRepo := repository.NewSessionRepository(redisClient)
	userRepo := repository.NewUserRepository(db)
	identityRepo := repository.NewOIDCIdentityRepository(db)
	userIdentityRepo := repository.NewUserWithIdentityRepository(db)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := appoidc.NewMockOIDCProvider(ctrl)
	mockProvider.EXPECT().
		BuildAuthorizationURL(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(state, nonce, codeChallenge string) string {
			return "https://example.com/auth?state=" + state
		})

	mockProviderWithLogin := appoidc.NewMockOIDCProviderWithLogin(ctrl)
	mockProviderWithLogin.EXPECT().
		ExchangeToken(gomock.Any(), "auth-code", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, code, codeVerifier, nonce string) (*appoidc.IDToken, error) {
			return &appoidc.IDToken{
				Subject: "sub-123",
				Name:    "Test User",
				Nonce:   nonce,
			}, nil
		})

	providerMap := map[domainoidc.ProviderID]appoidc.OIDCProvider{
		domainoidc.ProviderGoogle: mockProvider,
	}

	loginProviderMap := map[domainoidc.ProviderID]appoidc.OIDCProviderWithLogin{
		domainoidc.ProviderGoogle: mockProviderWithLogin,
	}

	sessionCfg := &sessioncfg.Config{
		Duration: time.Hour,
		Secret:   "super-secret",
	}
	jwtGenerator := jwt.NewSessionJWTGenerator(sessionCfg)
	jwtValidator := jwt.NewSessionJWTValidator(sessionCfg)

	paramsGenerator := appoidc.NewParamsGenerator(providerMap, paramsRepo)
	loginHandler := appoidc.NewLoginHandler(
		loginProviderMap,
		paramsRepo,
		sessionRepo,
		userRepo,
		identityRepo,
		userIdentityRepo,
		jwtGenerator,
		sessionCfg,
	)
	validateUseCase := appsession.NewValidateSessionHandler(sessionRepo, jwtValidator)
	logoutUseCase := applogout.NewLogoutHandler(sessionRepo, jwtValidator)

	service := authsvc.NewService(paramsGenerator, loginHandler, validateUseCase, logoutUseCase)

	paramsResp, err := service.OIDCParams(ctx, &authv1.OIDCParamsRequest{
		Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE,
	})
	if err != nil {
		t.Fatalf("OIDCParams returned error: %v", err)
	}

	storedParams, err := paramsRepo.GetParamsByState(ctx, paramsResp.GetState())
	if err != nil {
		t.Fatalf("failed to load stored params: %v", err)
	}

	mockProviderWithLogin.EXPECT().
		ProviderID().
		Return(domainoidc.ProviderGoogle).AnyTimes()

	mockProviderWithLogin.EXPECT().
		BuildAuthorizationURL(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes()

	loginResp, err := service.OIDCLogin(ctx, &authv1.OIDCLoginRequest{
		Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE,
		Code:     "auth-code",
		State:    storedParams.State(),
	})
	if err != nil {
		t.Fatalf("OIDCLogin returned error: %v", err)
	}

	validateResp, err := service.ValidateSession(ctx, &authv1.ValidateSessionRequest{
		SessionToken: loginResp.GetSessionToken(),
	})
	if err != nil {
		t.Fatalf("ValidateSession returned error: %v", err)
	}

	if validateResp.GetUserId() == "" {
		t.Fatalf("expected user id in validate response")
	}

	_, err = service.Logout(ctx, &authv1.LogoutRequest{
		SessionToken: loginResp.GetSessionToken(),
	})
	if err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}

	_, err = service.ValidateSession(ctx, &authv1.ValidateSessionRequest{
		SessionToken: loginResp.GetSessionToken(),
	})
	if err == nil {
		t.Fatalf("expected validation error after logout")
	}
}

func TestAuthE2EValidateInvalidSession(t *testing.T) {
	ctx := context.Background()

	redisClient, cleanupRedis := testutil.SetupRedisContainer(ctx, t)
	defer cleanupRedis()

	db, cleanupPostgres := testutil.SetupPostgresContainer(ctx, t)
	defer cleanupPostgres()

	if err := db.AutoMigrate(&repository.UserModel{}, &repository.OIDCIdentityModel{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	paramsRepo := repository.NewOIDCParamsRepository(redisClient)
	sessionRepo := repository.NewSessionRepository(redisClient)
	userRepo := repository.NewUserRepository(db)
	identityRepo := repository.NewOIDCIdentityRepository(db)
	userIdentityRepo := repository.NewUserWithIdentityRepository(db)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProvider := appoidc.NewMockOIDCProvider(ctrl)
	mockProvider.EXPECT().
		BuildAuthorizationURL(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes()

	mockProviderWithLogin := appoidc.NewMockOIDCProviderWithLogin(ctrl)
	mockProviderWithLogin.EXPECT().
		ProviderID().
		Return(domainoidc.ProviderGoogle).AnyTimes()
	mockProviderWithLogin.EXPECT().
		ExchangeToken(gomock.Any(), "code-invalid", gomock.Any(), gomock.Any()).
		Return(nil, appoidc.ErrCodeInvalid).AnyTimes()

	providerMap := map[domainoidc.ProviderID]appoidc.OIDCProvider{
		domainoidc.ProviderGoogle: mockProvider,
	}
	loginMap := map[domainoidc.ProviderID]appoidc.OIDCProviderWithLogin{
		domainoidc.ProviderGoogle: mockProviderWithLogin,
	}

	sessionCfg := &sessioncfg.Config{
		Duration: time.Minute,
		Secret:   "super-secret",
	}
	jwtGenerator := jwt.NewSessionJWTGenerator(sessionCfg)
	jwtValidator := jwt.NewSessionJWTValidator(sessionCfg)

	paramsGenerator := appoidc.NewParamsGenerator(providerMap, paramsRepo)
	loginHandler := appoidc.NewLoginHandler(
		loginMap,
		paramsRepo,
		sessionRepo,
		userRepo,
		identityRepo,
		userIdentityRepo,
		jwtGenerator,
		sessionCfg,
	)
	validateUseCase := appsession.NewValidateSessionHandler(sessionRepo, jwtValidator)
	logoutUseCase := applogout.NewLogoutHandler(sessionRepo, jwtValidator)
	service := authsvc.NewService(paramsGenerator, loginHandler, validateUseCase, logoutUseCase)

	_, err := service.OIDCParams(ctx, &authv1.OIDCParamsRequest{
		Provider: authv1.OIDCProvider_OIDC_PROVIDER_GOOGLE,
	})
	if err != nil {
		t.Fatalf("params failed: %v", err)
	}

	// invalid token
	_, err = service.ValidateSession(ctx, &authv1.ValidateSessionRequest{
		SessionToken: "bad-token",
	})
	if err == nil || !errors.Is(err, appsession.ErrSessionTokenInvalid) {
		t.Fatalf("expected invalid token error, got %v", err)
	}

	userID, err := user.NewID()
	if err != nil {
		t.Fatalf("failed to create user id: %v", err)
	}

	now := time.Now().UTC().Add(-2 * time.Minute)

	session, err := domainsession.NewSession(userID, now, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := sessionRepo.SaveSession(ctx, session); err == nil {
		t.Fatalf("expected save session to fail due to expiry")
	}

	expiredToken, err := jwtGenerator.Generate(session, user.NewUser(userID, user.MustColor("#000000")))
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = service.ValidateSession(ctx, &authv1.ValidateSessionRequest{
		SessionToken: expiredToken,
	})
	if err == nil || !errors.Is(err, appsession.ErrSessionTokenInvalid) {
		t.Fatalf("expected invalid token error for expired session, got %v", err)
	}
}
