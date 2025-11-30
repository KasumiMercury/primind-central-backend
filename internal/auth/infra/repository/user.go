package repository

import (
	"context"
	"errors"
	"time"

	domainuser "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/clock"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrUserRequired = errors.New("user is required")

type UserModel struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	Color     string    `gorm:"type:varchar(7);not null"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
}

func (UserModel) TableName() string {
	return "auth_users"
}

type userRepository struct {
	db    *gorm.DB
	clock clock.Clock
}

func NewUserRepository(db *gorm.DB) domainuser.UserRepository {
	return &userRepository{
		db:    db,
		clock: &clock.RealClock{},
	}
}

func (r *userRepository) SaveUser(ctx context.Context, u *domainuser.User) error {
	if u == nil {
		return ErrUserRequired
	}

	record := UserModel{
		ID:        u.ID().String(),
		Color:     u.Color().String(),
		CreatedAt: r.clock.Now(),
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&record).
		Error
}

func (r *userRepository) GetUserByID(ctx context.Context, id domainuser.ID) (*domainuser.User, error) {
	var record UserModel
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainuser.ErrUserNotFound
		}

		return nil, err
	}

	userID, err := domainuser.NewIDFromString(record.ID)
	if err != nil {
		return nil, err
	}

	color, err := domainuser.NewColor(record.Color)
	if err != nil {
		return nil, err
	}

	return domainuser.NewUser(userID, color), nil
}
