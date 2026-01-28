package task

import (
	"context"
	"errors"
	"log/slog"

	connect "connectrpc.com/connect"
	taskv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1"
	"github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1/taskv1connect"
	appperiod "github.com/KasumiMercury/primind-central-backend/internal/task/app/period"
	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/interceptor"
)

// PeriodSettingService implements the UserPeriodSettingsService
type PeriodSettingService struct {
	getPeriodSettings    appperiod.GetPeriodSettingsUseCase
	updatePeriodSettings appperiod.UpdatePeriodSettingsUseCase
	logger               *slog.Logger
}

var _ taskv1connect.UserPeriodSettingsServiceHandler = (*PeriodSettingService)(nil)

// NewPeriodSettingService creates a new PeriodSettingService
func NewPeriodSettingService(
	getPeriodSettingsUseCase appperiod.GetPeriodSettingsUseCase,
	updatePeriodSettingsUseCase appperiod.UpdatePeriodSettingsUseCase,
) *PeriodSettingService {
	return &PeriodSettingService{
		getPeriodSettings:    getPeriodSettingsUseCase,
		updatePeriodSettings: updatePeriodSettingsUseCase,
		logger:               slog.Default().With(slog.String("module", "task")).WithGroup("task").WithGroup("periodsetting"),
	}
}

// GetUserPeriodSettings retrieves user period settings
func (s *PeriodSettingService) GetUserPeriodSettings(
	ctx context.Context,
	req *taskv1.GetUserPeriodSettingsRequest,
) (*taskv1.GetUserPeriodSettingsResponse, error) {
	token := interceptor.ExtractSessionToken(ctx)
	if token == "" {
		s.logger.Warn("get period settings called without session token")

		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("session token required"))
	}

	result, err := s.getPeriodSettings.GetPeriodSettings(ctx, &appperiod.GetPeriodSettingsRequest{
		SessionToken: token,
	})
	if err != nil {
		switch {
		case errors.Is(err, appperiod.ErrUnauthorized):
			s.logger.Info("unauthorized get period settings attempt")

			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		case errors.Is(err, appperiod.ErrAuthServiceUnavailable):
			s.logger.Error("auth service unavailable during get period settings", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeUnavailable, err)
		default:
			s.logger.Error("unexpected get period settings error", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	s.logger.Info("period settings retrieved", slog.Int("custom_count", len(result.Settings)))

	return &taskv1.GetUserPeriodSettingsResponse{
		Settings: convertToProtoPeriodSettings(result.Settings),
		Defaults: convertToProtoPeriodSettings(result.Defaults),
	}, nil
}

// UpdateUserPeriodSettings updates user period settings
func (s *PeriodSettingService) UpdateUserPeriodSettings(
	ctx context.Context,
	req *taskv1.UpdateUserPeriodSettingsRequest,
) (*taskv1.UpdateUserPeriodSettingsResponse, error) {
	token := interceptor.ExtractSessionToken(ctx)
	if token == "" {
		s.logger.Warn("update period settings called without session token")

		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("session token required"))
	}

	// Convert proto settings to app layer settings
	settings := make([]appperiod.PeriodSettingItem, 0, len(req.GetSettings()))
	for _, ps := range req.GetSettings() {
		taskType, err := protoTaskTypeToDomain(ps.GetTaskType())
		if err != nil {
			s.logger.Warn("invalid task type in period settings", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		settings = append(settings, appperiod.PeriodSettingItem{
			TaskType:      taskType,
			PeriodMinutes: int(ps.GetPeriodMinutes()),
		})
	}

	result, err := s.updatePeriodSettings.UpdatePeriodSettings(ctx, &appperiod.UpdatePeriodSettingsRequest{
		SessionToken: token,
		Settings:     settings,
	})
	if err != nil {
		switch {
		case errors.Is(err, appperiod.ErrUnauthorized):
			s.logger.Info("unauthorized update period settings attempt")

			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		case errors.Is(err, appperiod.ErrAuthServiceUnavailable):
			s.logger.Error("auth service unavailable during update period settings", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeUnavailable, err)
		case errors.Is(err, appperiod.ErrScheduledTypeNotAllowed),
			errors.Is(err, appperiod.ErrInvalidPeriodMinutes),
			errors.Is(err, appperiod.ErrInvalidTaskType):
			s.logger.Warn("invalid update period settings request", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			s.logger.Error("unexpected update period settings error", slog.String("error", err.Error()))

			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	s.logger.Info("period settings updated", slog.Int("count", len(result.Settings)))

	return &taskv1.UpdateUserPeriodSettingsResponse{
		Settings: convertToProtoPeriodSettings(result.Settings),
	}, nil
}

func convertToProtoPeriodSettings(items []appperiod.PeriodSettingItem) []*taskv1.PeriodSetting {
	result := make([]*taskv1.PeriodSetting, 0, len(items))
	for _, item := range items {
		result = append(result, &taskv1.PeriodSetting{
			TaskType:      domainTaskTypeToProto(item.TaskType),
			PeriodMinutes: int64(item.PeriodMinutes),
		})
	}

	return result
}

func protoTaskTypeToDomain(taskType taskv1.TaskType) (domaintask.Type, error) {
	switch taskType {
	case taskv1.TaskType_TASK_TYPE_SHORT:
		return domaintask.TypeShort, nil
	case taskv1.TaskType_TASK_TYPE_NEAR:
		return domaintask.TypeNear, nil
	case taskv1.TaskType_TASK_TYPE_RELAXED:
		return domaintask.TypeRelaxed, nil
	case taskv1.TaskType_TASK_TYPE_SCHEDULED:
		return "", errors.New("scheduled task type is not allowed for period settings")
	case taskv1.TaskType_TASK_TYPE_UNSPECIFIED:
		return "", errors.New("task type is required")
	default:
		return "", errors.New("unsupported task type")
	}
}

func domainTaskTypeToProto(taskType domaintask.Type) taskv1.TaskType {
	switch taskType {
	case domaintask.TypeShort:
		return taskv1.TaskType_TASK_TYPE_SHORT
	case domaintask.TypeNear:
		return taskv1.TaskType_TASK_TYPE_NEAR
	case domaintask.TypeRelaxed:
		return taskv1.TaskType_TASK_TYPE_RELAXED
	case domaintask.TypeScheduled:
		return taskv1.TaskType_TASK_TYPE_SCHEDULED
	default:
		return taskv1.TaskType_TASK_TYPE_UNSPECIFIED
	}
}
