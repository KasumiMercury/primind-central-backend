package repository

import (
	"context"
	"errors"

	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	domainidentity "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/clock"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrOIDCIdentityConflict = errors.New("oidc identity belongs to a different user")

type userWithIdentityRepository struct {
	db    *gorm.DB
	clock clock.Clock
}

func newUserWithIdentityRepository(db *gorm.DB, clk clock.Clock) appoidc.UserWithOIDCIdentityRepository {
	return &userWithIdentityRepository{
		db:    db,
		clock: clk,
	}
}

func NewUserWithIdentityRepository(db *gorm.DB) appoidc.UserWithOIDCIdentityRepository {
	return newUserWithIdentityRepository(db, &clock.RealClock{})
}

func NewUserWithIdentityRepositoryWithClock(db *gorm.DB, clk clock.Clock) appoidc.UserWithOIDCIdentityRepository {
	return newUserWithIdentityRepository(db, clk)
}

func (r *userWithIdentityRepository) SaveUserWithOIDCIdentity(
	ctx context.Context,
	u *domainuser.User,
	identity *domainidentity.OIDCIdentity,
) error {
	if u == nil {
		return ErrUserRequired
	}

	if identity == nil {
		return ErrIdentityRequired
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userRecord := UserModel{
			ID:        u.ID().String(),
			Color:     u.Color().String(),
			CreatedAt: r.clock.Now(),
		}

		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&userRecord).Error; err != nil {
			return err
		}

		var existingIdentity OIDCIdentityModel

		identityLookup := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("provider = ? AND subject = ?", identity.Provider(), identity.Subject()).
			First(&existingIdentity)

		if identityLookup.Error != nil && !errors.Is(identityLookup.Error, gorm.ErrRecordNotFound) {
			return identityLookup.Error
		}

		if identityLookup.Error == nil {
			if existingIdentity.UserID != identity.UserID().String() {
				return ErrOIDCIdentityConflict
			}

			return nil
		}

		identityRecord := OIDCIdentityModel{
			UserID:    identity.UserID().String(),
			Provider:  string(identity.Provider()),
			Subject:   identity.Subject(),
			CreatedAt: r.clock.Now(),
		}

		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "subject"}},
			DoNothing: true,
		}).Create(&identityRecord)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			check := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("provider = ? AND subject = ?", identity.Provider(), identity.Subject()).
				First(&existingIdentity)
			if check.Error != nil {
				return check.Error
			}

			if existingIdentity.UserID != identity.UserID().String() {
				return ErrOIDCIdentityConflict
			}
		}

		return nil
	})
}
