package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/testutil"
)

func setupTestRepositories(t *testing.T) Repositories {
	t.Helper()

	ctx := context.Background()

	redisClient, cleanupRedis := testutil.SetupRedisContainer(ctx, t)
	t.Cleanup(cleanupRedis)

	db, cleanupPostgres := testutil.SetupPostgresContainer(ctx, t)
	t.Cleanup(cleanupPostgres)

	if err := db.AutoMigrate(&repository.UserModel{}, &repository.OIDCIdentityModel{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	return Repositories{
		Params:       repository.NewOIDCParamsRepository(redisClient),
		Sessions:     repository.NewSessionRepository(redisClient),
		Users:        repository.NewUserRepository(db),
		OIDCIdentity: repository.NewOIDCIdentityRepository(db),
		UserIdentity: repository.NewUserWithIdentityRepository(db),
	}
}

func clearOIDCEnv(t *testing.T) {
	t.Helper()

	t.Setenv("OIDC_GOOGLE_CLIENT_ID", "")
	t.Setenv("OIDC_GOOGLE_CLIENT_SECRET", "")
	t.Setenv("OIDC_GOOGLE_REDIRECT_URI", "")
}

func TestNewHTTPHandlerSuccess(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")
	t.Setenv("SESSION_DURATION", "1h")
	clearOIDCEnv(t)

	repos := setupTestRepositories(t)

	path, handler, err := NewHTTPHandler(context.Background(), repos)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if path == "" {
		t.Fatalf("expected path to be set")
	}

	if handler == nil {
		t.Fatalf("expected handler to be non-nil")
	}
}

func TestNewHTTPHandlerError(t *testing.T) {
	tests := []struct {
		name      string
		setupEnv  func()
		repos     Repositories
		wantError bool
	}{
		{
			name: "missing session secret",
			setupEnv: func() {
				clearOIDCEnv(t)
				os.Unsetenv("SESSION_SECRET")
			},
			repos:     Repositories{},
			wantError: true,
		},
		{
			name: "incomplete repositories",
			setupEnv: func() {
				t.Setenv("SESSION_SECRET", "secret")
				t.Setenv("SESSION_DURATION", "1h")
				clearOIDCEnv(t)
			},
			repos: Repositories{
				Params: nil,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			_, _, err := NewHTTPHandler(ctx, tt.repos)
			if tt.wantError && err == nil {
				t.Fatalf("expected error but got nil")
			}

			if !tt.wantError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
