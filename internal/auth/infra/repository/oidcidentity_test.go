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

func setupIdentityDB(t *testing.T) *gorm.DB {
	t.Helper()

	ctx := context.Background()
	db, cleanup := testutil.SetupPostgresContainer(ctx, t)
	t.Cleanup(cleanup)

	if err := db.AutoMigrate(&UserModel{}, &OIDCIdentityModel{}); err != nil {
		t.Fatalf("failed to migrate identity tables: %v", err)
	}

	return db
}

func TestOIDCIdentityRepositoryIntegrationSuccess(t *testing.T) {
	db := setupIdentityDB(t)
	identityRepo := &oidcIdentityRepository{
		db:    db,
		clock: clock.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}
	userRepo := &userRepository{
		db:    db,
		clock: clock.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	userID, _ := domainuser.NewID()
	color := domainuser.MustColor("#abcdef")

	u := domainuser.NewUser(userID, color)
	if err := userRepo.SaveUser(context.Background(), u); err != nil {
		t.Fatalf("SaveUser returned error: %v", err)
	}

	identity, _ := domainidentity.NewOIDCIdentity(userID, domainoidc.ProviderGoogle, "subject-1")
	if err := identityRepo.SaveOIDCIdentity(context.Background(), identity); err != nil {
		t.Fatalf("SaveOIDCIdentity returned error: %v", err)
	}

	found, err := identityRepo.GetOIDCIdentityByProviderSubject(context.Background(), domainoidc.ProviderGoogle, "subject-1")
	if err != nil {
		t.Fatalf("GetOIDCIdentityByProviderSubject returned error: %v", err)
	}

	if found.UserID() != userID {
		t.Fatalf("expected user id %s, got %s", userID.String(), found.UserID().String())
	}
}

func TestOIDCIdentityRepositoryIntegrationError(t *testing.T) {
	db := setupIdentityDB(t)
	identityRepo := NewOIDCIdentityRepository(db)

	if err := identityRepo.SaveOIDCIdentity(context.Background(), nil); !errors.Is(err, ErrIdentityRequired) {
		t.Fatalf("expected ErrIdentityRequired, got %v", err)
	}

	if _, err := identityRepo.GetOIDCIdentityByProviderSubject(context.Background(), domainoidc.ProviderGoogle, "missing"); !errors.Is(err, domainidentity.ErrOIDCIdentityNotFound) {
		t.Fatalf("expected ErrOIDCIdentityNotFound, got %v", err)
	}
}

func TestOIDCIdentityRepositoryWithFixedClock(t *testing.T) {
	db := setupIdentityDB(t)

	fixedTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	identityRepo := NewOIDCIdentityRepositoryWithClock(db, clock.NewFixedClock(fixedTime))
	userRepo := NewUserRepositoryWithClock(db, clock.NewFixedClock(fixedTime))

	userID, _ := domainuser.NewID()
	color := domainuser.MustColor("#123456")

	u := domainuser.NewUser(userID, color)
	if err := userRepo.SaveUser(context.Background(), u); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	identity, err := domainidentity.NewOIDCIdentity(userID, domainoidc.ProviderGoogle, "subject-123")
	if err != nil {
		t.Fatalf("failed to create identity: %v", err)
	}

	if err := identityRepo.SaveOIDCIdentity(context.Background(), identity); err != nil {
		t.Fatalf("SaveOIDCIdentity with fixed clock failed: %v", err)
	}

	var record OIDCIdentityModel
	if err := db.First(&record, "provider = ? AND subject = ?", domainoidc.ProviderGoogle, "subject-123").Error; err != nil {
		t.Fatalf("failed to query identity record: %v", err)
	}

	if !record.CreatedAt.Equal(fixedTime) {
		t.Fatalf("expected CreatedAt to be %v, got %v", fixedTime, record.CreatedAt)
	}
}
