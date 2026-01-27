package repository

import (
	"context"
	"errors"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/period"
	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PeriodSettingModel is the GORM model for user period settings
type PeriodSettingModel struct {
	UserID    string                             `gorm:"type:uuid;primaryKey"`
	Periods   datatypes.JSONType[map[string]int] `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time                          `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time                          `gorm:"not null;autoUpdateTime"`
}

// TableName returns the table name for the model
func (PeriodSettingModel) TableName() string {
	return "user_period_settings"
}

type periodSettingRepository struct {
	db *gorm.DB
}

// NewPeriodSettingRepository creates a new period setting repository
func NewPeriodSettingRepository(db *gorm.DB) period.PeriodSettingRepository {
	return &periodSettingRepository{db: db}
}

// GetByUserID retrieves the period settings for a user
func (r *periodSettingRepository) GetByUserID(ctx context.Context, userID user.ID) (*period.UserPeriodSettings, error) {
	var record PeriodSettingModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID.String()).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return empty settings if not found
			return period.NewUserPeriodSettings(userID, nil)
		}

		return nil, err
	}

	return r.recordToSettings(userID, record)
}

// Save creates or updates the period settings for a user
func (r *periodSettingRepository) Save(ctx context.Context, settings *period.UserPeriodSettings) error {
	if settings == nil {
		return ErrPeriodSettingRequired
	}

	periodsMap := r.settingsToPeriodsMap(settings)

	record := PeriodSettingModel{
		UserID:  settings.UserID().String(),
		Periods: datatypes.NewJSONType(periodsMap),
	}

	// Upsert: create if not exists, update if exists
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"periods", "updated_at"}),
		}).
		Create(&record).Error
}

func (r *periodSettingRepository) recordToSettings(userID user.ID, record PeriodSettingModel) (*period.UserPeriodSettings, error) {
	periods := make(map[task.Type]int)

	for key, value := range record.Periods.Data() {
		taskType, err := task.NewType(key)
		if err != nil {
			continue // Skip invalid task types
		}

		periods[taskType] = value
	}

	return period.NewUserPeriodSettings(userID, periods)
}

func (r *periodSettingRepository) settingsToPeriodsMap(settings *period.UserPeriodSettings) map[string]int {
	result := make(map[string]int)

	for taskType, minutes := range settings.Periods() {
		result[string(taskType)] = minutes
	}

	return result
}
