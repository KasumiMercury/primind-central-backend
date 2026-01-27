package periodsetting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/period"
	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
)

// PeriodSettingItem represents a single period setting
type PeriodSettingItem struct {
	TaskType      task.Type
	PeriodMinutes int
}

// GetPeriodSettingsRequest is the request for getting period settings
type GetPeriodSettingsRequest struct {
	SessionToken string
}

// GetPeriodSettingsResult is the result of getting period settings
type GetPeriodSettingsResult struct {
	Settings []PeriodSettingItem
	Defaults []PeriodSettingItem
}

// UpdatePeriodSettingsRequest is the request for updating period settings
type UpdatePeriodSettingsRequest struct {
	SessionToken string
	Settings     []PeriodSettingItem
}

// UpdatePeriodSettingsResult is the result of updating period settings
type UpdatePeriodSettingsResult struct {
	Settings []PeriodSettingItem
}

// GetPeriodSettingsUseCase defines the interface for getting period settings
type GetPeriodSettingsUseCase interface {
	GetPeriodSettings(ctx context.Context, req *GetPeriodSettingsRequest) (*GetPeriodSettingsResult, error)
}

// UpdatePeriodSettingsUseCase defines the interface for updating period settings
type UpdatePeriodSettingsUseCase interface {
	UpdatePeriodSettings(ctx context.Context, req *UpdatePeriodSettingsRequest) (*UpdatePeriodSettingsResult, error)
}

type getPeriodSettingsHandler struct {
	authClient authclient.AuthClient
	periodRepo period.PeriodSettingRepository
	logger     *slog.Logger
}

// NewGetPeriodSettingsHandler creates a new handler for getting period settings
func NewGetPeriodSettingsHandler(
	authClient authclient.AuthClient,
	periodRepo period.PeriodSettingRepository,
) GetPeriodSettingsUseCase {
	return &getPeriodSettingsHandler{
		authClient: authClient,
		periodRepo: periodRepo,
		logger:     slog.Default().With(slog.String("module", "task")).WithGroup("task").WithGroup("getperiod"),
	}
}

func (h *getPeriodSettingsHandler) GetPeriodSettings(ctx context.Context, req *GetPeriodSettingsRequest) (*GetPeriodSettingsResult, error) {
	if req == nil {
		return nil, ErrGetPeriodSettingsRequestRequired
	}

	userIDstr, err := h.authClient.ValidateSession(ctx, req.SessionToken)
	if err != nil {
		if errors.Is(err, authclient.ErrUnauthorized) {
			h.logger.Info("session validation failed", slog.String("error", err.Error()))

			return nil, ErrUnauthorized
		}

		h.logger.Error("session validation failed", slog.String("error", err.Error()))

		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	userID, err := domainuser.NewIDFromString(userIDstr)
	if err != nil {
		h.logger.Warn("invalid user ID format", slog.String("error", err.Error()))

		return nil, err
	}

	settings, err := h.periodRepo.GetByUserID(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get period settings", slog.String("error", err.Error()))

		return nil, err
	}

	// Convert to result
	settingsItems := make([]PeriodSettingItem, 0)
	for taskType, minutes := range settings.Periods() {
		settingsItems = append(settingsItems, PeriodSettingItem{
			TaskType:      taskType,
			PeriodMinutes: minutes,
		})
	}

	// Get defaults
	defaults := period.DefaultPeriodSettings()

	defaultItems := make([]PeriodSettingItem, 0, len(defaults))
	for taskType, minutes := range defaults {
		defaultItems = append(defaultItems, PeriodSettingItem{
			TaskType:      taskType,
			PeriodMinutes: minutes,
		})
	}

	h.logger.Info("period settings retrieved", slog.Int("custom_count", len(settingsItems)))

	return &GetPeriodSettingsResult{
		Settings: settingsItems,
		Defaults: defaultItems,
	}, nil
}

type updatePeriodSettingsHandler struct {
	authClient authclient.AuthClient
	periodRepo period.PeriodSettingRepository
	logger     *slog.Logger
}

// NewUpdatePeriodSettingsHandler creates a new handler for updating period settings
func NewUpdatePeriodSettingsHandler(
	authClient authclient.AuthClient,
	periodRepo period.PeriodSettingRepository,
) UpdatePeriodSettingsUseCase {
	return &updatePeriodSettingsHandler{
		authClient: authClient,
		periodRepo: periodRepo,
		logger:     slog.Default().With(slog.String("module", "task")).WithGroup("task").WithGroup("updateperiod."),
	}
}

func (h *updatePeriodSettingsHandler) UpdatePeriodSettings(ctx context.Context, req *UpdatePeriodSettingsRequest) (*UpdatePeriodSettingsResult, error) {
	if req == nil {
		return nil, ErrUpdatePeriodSettingsRequestRequired
	}

	userIDstr, err := h.authClient.ValidateSession(ctx, req.SessionToken)
	if err != nil {
		if errors.Is(err, authclient.ErrUnauthorized) {
			h.logger.Info("session validation failed", slog.String("error", err.Error()))

			return nil, ErrUnauthorized
		}

		h.logger.Error("session validation failed", slog.String("error", err.Error()))

		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	userID, err := domainuser.NewIDFromString(userIDstr)
	if err != nil {
		h.logger.Warn("invalid user ID format", slog.String("error", err.Error()))

		return nil, err
	}

	// Convert request items to periods map
	periods := make(map[task.Type]int)
	for _, item := range req.Settings {
		periods[item.TaskType] = item.PeriodMinutes
	}

	// Create domain object (this validates the input)
	settings, err := period.NewUserPeriodSettings(userID, periods)
	if err != nil {
		h.logger.Warn("invalid period settings", slog.String("error", err.Error()))

		return nil, err
	}

	// Save settings
	if err := h.periodRepo.Save(ctx, settings); err != nil {
		h.logger.Error("failed to save period settings", slog.String("error", err.Error()))

		return nil, err
	}

	// Convert to result
	resultItems := make([]PeriodSettingItem, 0, len(req.Settings))
	for taskType, minutes := range settings.Periods() {
		resultItems = append(resultItems, PeriodSettingItem{
			TaskType:      taskType,
			PeriodMinutes: minutes,
		})
	}

	h.logger.Info("period settings updated", slog.Int("count", len(resultItems)))

	return &UpdatePeriodSettingsResult{
		Settings: resultItems,
	}, nil
}
