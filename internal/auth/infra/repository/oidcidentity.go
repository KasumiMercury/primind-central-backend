package repository

import (
	"context"
	"errors"
	"time"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	domainidentity "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrIdentityRequired = errors.New("identity is required")

type OIDCIdentityModel struct {
	UserID    string    `gorm:"type:uuid;not null;index"`
	User      UserModel `gorm:"constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:UserID;references:ID"`
	Provider  string    `gorm:"type:text;not null;primaryKey"`
	Subject   string    `gorm:"type:text;not null;primaryKey"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
}

func (OIDCIdentityModel) TableName() string {
	return "auth_oidc_identities"
}

type oidcIdentityRepository struct {
	db *gorm.DB
}

func NewOIDCIdentityRepository(db *gorm.DB) domainidentity.OIDCIdentityRepository {
	return &oidcIdentityRepository{db: db}
}

func (r *oidcIdentityRepository) SaveOIDCIdentity(ctx context.Context, identity *domainidentity.OIDCIdentity) error {
	if identity == nil {
		return ErrIdentityRequired
	}

	record := OIDCIdentityModel{
		UserID:    identity.UserID().String(),
		Provider:  string(identity.Provider()),
		Subject:   identity.Subject(),
		CreatedAt: time.Now().UTC(),
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&record).
		Error
}

func (r *oidcIdentityRepository) GetOIDCIdentityByProviderSubject(
	ctx context.Context,
	provider domainoidc.ProviderID,
	subject string,
) (*domainidentity.OIDCIdentity, error) {
	var record OIDCIdentityModel
	if err := r.db.WithContext(ctx).
		Where("provider = ? AND subject = ?", provider, subject).
		First(&record).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainidentity.ErrOIDCIdentityNotFound
		}

		return nil, err
	}

	userID, err := user.NewIDFromString(record.UserID)
	if err != nil {
		return nil, err
	}

	return domainidentity.NewOIDCIdentity(userID, domainoidc.ProviderID(record.Provider), record.Subject)
}
