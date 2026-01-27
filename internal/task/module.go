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
	"github.com/KasumiMercury/primind-central-backend/internal/task/config"
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

func NewHTTPHandler(
	ctx context.Context,
	taskRepo domaintask.TaskRepository,
	taskArchiveRepo domaintask.TaskArchiveRepository,
	cfg *config.Config,
) (path string, handler http.Handler, err error) {
	logger := slog.Default().With(
		slog.String("module", string(moduleName)),
	).WithGroup("task")

	remindQueue, cancelRemindQueue, client, err := NewRemindQueues(ctx, &cfg.TaskQueue)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create remind queues: %w", err)
	}

	defer func() {
		if err != nil {
			if closeErr := client.Close(); closeErr != nil {
				logger.Warn("failed to close task queue client during cleanup", slog.String("error", closeErr.Error()))
			}
		}
	}()

	return NewHTTPHandlerWithRepositories(ctx, Repositories{
		Tasks:               taskRepo,
		TaskArchive:         taskArchiveRepo,
		AuthClient:          authclient.NewAuthClient(cfg.AuthServiceURL),
		DeviceClient:        deviceclient.NewDeviceClient(cfg.DeviceServiceURL),
		RemindRegisterQueue: remindQueue,
		RemindCancelQueue:   cancelRemindQueue,
		TaskQueueClient:     client,
	})
}

func NewHTTPHandlerWithRepositories(ctx context.Context, repos Repositories) (string, http.Handler, error) {
	logger := slog.Default().With(
		slog.String("module", string(moduleName)),
	).WithGroup("task")

	logger.Debug("initializing task module")

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

	// Create OpenTelemetry interceptor for tracing
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		logger.Error("failed to create otelconnect interceptor", slog.String("error", err.Error()))

		return "", nil, fmt.Errorf("failed to create otelconnect interceptor: %w", err)
	}

	// Common interceptor options
	interceptorOpts := connect.WithInterceptors(
		otelInterceptor,
		middleware.ConnectLoggingInterceptor(moduleName),
		interceptor.AuthInterceptor(),
	)

	// Create HTTP mux for multiple services
	mux := http.NewServeMux()

	// Register TaskService
	taskPath, taskHandler := taskv1connect.NewTaskServiceHandler(taskService, interceptorOpts)
	mux.Handle(taskPath, taskHandler)
	logger.Info("task service handler registered", slog.String("path", taskPath))

	// Register UserPeriodSettingsService if PeriodSettings repository is configured
	if repos.PeriodSettings != nil {
		getPeriodSettingsUseCase := appperiodsetting.NewGetPeriodSettingsHandler(repos.AuthClient, repos.PeriodSettings)
		updatePeriodSettingsUseCase := appperiodsetting.NewUpdatePeriodSettingsHandler(repos.AuthClient, repos.PeriodSettings)
		periodSettingService := tasksvc.NewPeriodSettingService(getPeriodSettingsUseCase, updatePeriodSettingsUseCase)

		periodSettingPath, periodSettingHandler := taskv1connect.NewUserPeriodSettingsServiceHandler(periodSettingService, interceptorOpts)
		mux.Handle(periodSettingPath, periodSettingHandler)
		logger.Info("period setting service handler registered", slog.String("path", periodSettingPath))
	}

	return "/task.v1.", mux, nil
}
