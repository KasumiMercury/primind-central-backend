package period

//go:generate mockgen -source=period_setting_repository.go -destination=mock_period_setting_repository.go -package=period

import (
	"context"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
)

// PeriodSettingRepository defines the interface for period setting persistence
type PeriodSettingRepository interface {
	// GetByUserID retrieves the period settings for a user
	// Returns an empty UserPeriodSettings if no settings exist
	GetByUserID(ctx context.Context, userID user.ID) (*UserPeriodSettings, error)

	// Save creates or updates the period settings for a user
	Save(ctx context.Context, settings *UserPeriodSettings) error
}
