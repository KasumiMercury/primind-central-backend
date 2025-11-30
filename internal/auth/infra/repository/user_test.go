package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	domainuser "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/clock"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/testutil"
	"gorm.io/gorm"
)

func setupUserDB(t *testing.T) *gorm.DB {
	t.Helper()

	ctx := context.Background()
	db, cleanup := testutil.SetupPostgresContainer(ctx, t)
	t.Cleanup(cleanup)

	if err := db.AutoMigrate(&UserModel{}); err != nil {
		t.Fatalf("failed to migrate user table: %v", err)
	}

	return db
}

func TestUserRepositoryIntegrationSuccess(t *testing.T) {
	db := setupUserDB(t)

	repo := &userRepository{
		db:    db,
		clock: clock.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	userID, _ := domainuser.NewID()
	color := domainuser.MustColor("#123abc")
	u := domainuser.NewUser(userID, color)

	if err := repo.SaveUser(context.Background(), u); err != nil {
		t.Fatalf("SaveUser returned error: %v", err)
	}

	found, err := repo.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserByID returned error: %v", err)
	}

	if found.ID() != userID || found.Color().String() != color.String() {
		t.Fatalf("unexpected user data")
	}
}

func TestUserRepositoryIntegrationError(t *testing.T) {
	db := setupUserDB(t)

	repo := NewUserRepository(db)

	if err := repo.SaveUser(context.Background(), nil); !errors.Is(err, ErrUserRequired) {
		t.Fatalf("expected ErrUserRequired, got %v", err)
	}

	missing, _ := domainuser.NewID()
	if _, err := repo.GetUserByID(context.Background(), missing); !errors.Is(err, domainuser.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
