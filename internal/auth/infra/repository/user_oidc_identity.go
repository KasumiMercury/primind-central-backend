package repository

import (
	"context"
	"time"

	appoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	domainidentity "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

		identityRecord := OIDCIdentityModel{
			UserID:    identity.UserID().String(),
			Provider:  string(identity.Provider()),
			Subject:   identity.Subject(),
			CreatedAt: time.Now().UTC(),
		}

		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&identityRecord).Error; err != nil {
			return err
		}

		return nil
	})
}
