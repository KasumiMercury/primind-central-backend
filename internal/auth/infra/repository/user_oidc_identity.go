package repository

import (
	"context"
	"errors"
	"time"

	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	domainidentity "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrOIDCIdentityConflict = errors.New("oidc identity belongs to a different user")

type userWithIdentityRepository struct {
	db *gorm.DB
}

func NewUserWithIdentityRepository(db *gorm.DB) appoidc.UserWithOIDCIdentityRepository {
	return &userWithIdentityRepository{db: db}
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
			CreatedAt: time.Now().UTC(),
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
			CreatedAt: time.Now().UTC(),
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
