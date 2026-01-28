package task

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1/taskv1connect"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/logging"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/middleware"
	appperiodsetting "github.com/KasumiMercury/primind-central-backend/internal/task/app/period"
	apptask "github.com/KasumiMercury/primind-central-backend/internal/task/app/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/period"
	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/deviceclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/interceptor"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindcancel"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindregister"
	tasksvc "github.com/KasumiMercury/primind-central-backend/internal/task/infra/service"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/taskqueue"
)

const moduleName logging.Module = "task"

type Repositories struct {
	Tasks               domaintask.TaskRepository
	TaskArchive         domaintask.TaskArchiveRepository
	PeriodSettings      period.PeriodSettingRepository
	AuthClient          authclient.AuthClient
	DeviceClient        deviceclient.DeviceClient
	RemindRegisterQueue remindregister.Queue
	RemindCancelQueue   remindcancel.Queue
	TaskQueueClient     taskqueue.Client
}

func (r *Repositories) Close() error {
	if r == nil || r.TaskQueueClient == nil {
		return nil
	}

	return r.TaskQueueClient.Close()
}

// newInterceptorOptions creates common Connect interceptor options for task module services.
func newInterceptorOptions() (connect.HandlerOption, error) {
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return nil, fmt.Errorf("failed to create otelconnect interceptor: %w", err)
	}

	return connect.WithInterceptors(
		otelInterceptor,
		middleware.ConnectLoggingInterceptor(moduleName),
		interceptor.AuthInterceptor(),
	), nil
}

// NewTaskServiceHandler creates and returns the TaskService HTTP handler.
// It returns the service path, handler, and any initialization error.
func NewTaskServiceHandler(ctx context.Context, repos Repositories) (string, http.Handler, error) {
	logger := slog.Default().With(
		slog.String("module", string(moduleName)),
	).WithGroup("task")

	logger.Debug("initializing task service")

	if repos.Tasks == nil {
		return "", nil, fmt.Errorf("task repository is not configured")
	}

	if repos.AuthClient == nil {
		return "", nil, fmt.Errorf("auth client is not configured")
	}

	if repos.DeviceClient == nil {
		return "", nil, fmt.Errorf("device client is not configured")
	}

	if repos.RemindRegisterQueue == nil {
		return "", nil, fmt.Errorf("remind register queue is not configured")
	}

	if repos.RemindCancelQueue == nil {
		return "", nil, fmt.Errorf("remind cancel queue is not configured")
	}

	if repos.TaskArchive == nil {
		return "", nil, fmt.Errorf("task archive repository is not configured")
	}

	createTaskUseCase := apptask.NewCreateTaskHandler(repos.AuthClient, repos.DeviceClient, repos.Tasks, repos.PeriodSettings, repos.RemindRegisterQueue)
	getTaskUseCase := apptask.NewGetTaskHandler(repos.AuthClient, repos.Tasks)
	listActiveTasksUseCase := apptask.NewListActiveTasksHandler(repos.AuthClient, repos.Tasks)
	updateTaskUseCase := apptask.NewUpdateTaskHandler(repos.AuthClient, repos.Tasks, repos.TaskArchive, repos.RemindCancelQueue)
	deleteTaskUseCase := apptask.NewDeleteTaskHandler(repos.AuthClient, repos.Tasks, repos.RemindCancelQueue)

	taskService := tasksvc.NewService(createTaskUseCase, getTaskUseCase, listActiveTasksUseCase, updateTaskUseCase, deleteTaskUseCase)

	interceptorOpts, err := newInterceptorOptions()
	if err != nil {
		logger.Error("failed to create interceptor options", slog.String("error", err.Error()))
		return "", nil, err
	}

	taskPath, taskHandler := taskv1connect.NewTaskServiceHandler(taskService, interceptorOpts)
	logger.Info("task service handler registered", slog.String("path", taskPath))

	return taskPath, taskHandler, nil
}

// NewPeriodSettingsServiceHandler creates and returns the UserPeriodSettingsService HTTP handler.
// It returns the service path, handler, and any initialization error.
func NewPeriodSettingsServiceHandler(ctx context.Context, repos Repositories) (string, http.Handler, error) {
	logger := slog.Default().With(
		slog.String("module", string(moduleName)),
	).WithGroup("period_setting")

	logger.Debug("initializing period settings service")

	if repos.AuthClient == nil {
		return "", nil, fmt.Errorf("auth client is not configured")
	}

	if repos.PeriodSettings == nil {
		return "", nil, fmt.Errorf("period settings repository is not configured")
	}

	getPeriodSettingsUseCase := appperiodsetting.NewGetPeriodSettingsHandler(repos.AuthClient, repos.PeriodSettings)
	updatePeriodSettingsUseCase := appperiodsetting.NewUpdatePeriodSettingsHandler(repos.AuthClient, repos.PeriodSettings)
	periodSettingService := tasksvc.NewPeriodSettingService(getPeriodSettingsUseCase, updatePeriodSettingsUseCase)

	interceptorOpts, err := newInterceptorOptions()
	if err != nil {
		logger.Error("failed to create interceptor options", slog.String("error", err.Error()))
		return "", nil, err
	}

	periodSettingPath, periodSettingHandler := taskv1connect.NewUserPeriodSettingsServiceHandler(periodSettingService, interceptorOpts)
	logger.Info("period setting service handler registered", slog.String("path", periodSettingPath))

	return periodSettingPath, periodSettingHandler, nil
}
