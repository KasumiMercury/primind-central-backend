package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	domainidentity "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/clock"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/testutil"
	"gorm.io/gorm"
)

func setupUserWithIdentityDB(t *testing.T) *gorm.DB {
	t.Helper()

	ctx := context.Background()
	db, cleanup := testutil.SetupPostgresContainer(ctx, t)
	t.Cleanup(cleanup)

	if err := db.AutoMigrate(&UserModel{}, &OIDCIdentityModel{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	return db
}

func TestUserWithIdentityRepositoryIntegrationSuccess(t *testing.T) {
	db := setupUserWithIdentityDB(t)
	repo := &userWithIdentityRepository{
		db:    db,
		clock: clock.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}
	identityRepo := &oidcIdentityRepository{
		db:    db,
		clock: clock.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	userID, _ := domainuser.NewID()
	userColor := domainuser.MustColor("#fff001")
	u := domainuser.NewUser(userID, userColor)
	identity, _ := domainidentity.NewOIDCIdentity(userID, domainoidc.ProviderGoogle, "subject-123")

	if err := repo.SaveUserWithOIDCIdentity(context.Background(), u, identity); err != nil {
		t.Fatalf("SaveUserWithOIDCIdentity returned error: %v", err)
	}

	found, err := identityRepo.GetOIDCIdentityByProviderSubject(context.Background(), domainoidc.ProviderGoogle, "subject-123")
	if err != nil {
		t.Fatalf("failed to fetch stored identity: %v", err)
	}

	if found.UserID() != userID {
		t.Fatalf("expected user id %s, got %s", userID.String(), found.UserID().String())
	}
}

func TestUserWithIdentityRepositoryIntegrationError(t *testing.T) {
	db := setupUserWithIdentityDB(t)
	repo := NewUserWithIdentityRepository(db)

	user1ID, _ := domainuser.NewID()
	user2ID, _ := domainuser.NewID()

	user1 := domainuser.NewUser(user1ID, domainuser.MustColor("#aaaaaa"))
	user2 := domainuser.NewUser(user2ID, domainuser.MustColor("#bbbbbb"))

	identityUser1, _ := domainidentity.NewOIDCIdentity(user1ID, domainoidc.ProviderGoogle, "duplicate-subject")
	if err := repo.SaveUserWithOIDCIdentity(context.Background(), user1, identityUser1); err != nil {
		t.Fatalf("failed to save initial identity: %v", err)
	}

	conflictingIdentity, _ := domainidentity.NewOIDCIdentity(user2ID, domainoidc.ProviderGoogle, "duplicate-subject")
	if err := repo.SaveUserWithOIDCIdentity(context.Background(), user2, conflictingIdentity); !errors.Is(err, ErrOIDCIdentityConflict) {
		t.Fatalf("expected ErrOIDCIdentityConflict, got %v", err)
	}
}
