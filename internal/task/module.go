package task

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	connect "connectrpc.com/connect"
	"github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1/taskv1connect"
	apptask "github.com/KasumiMercury/primind-central-backend/internal/task/app/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/config"
	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/deviceclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/interceptor"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindcancel"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindregister"
	tasksvc "github.com/KasumiMercury/primind-central-backend/internal/task/infra/service"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/taskqueue"
)

type Repositories struct {
	Tasks               domaintask.TaskRepository
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
	cfg *config.Config,
) (path string, handler http.Handler, err error) {
	remindQueue, cancelRemindQueue, client, err := NewRemindQueues(ctx, &cfg.TaskQueue)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create remind queues: %w", err)
	}

	defer func() {
		if err != nil {
			if closeErr := client.Close(); closeErr != nil {
				slog.Warn("failed to close task queue client during cleanup", slog.String("error", closeErr.Error()))
			}
		}
	}()

	return NewHTTPHandlerWithRepositories(ctx, Repositories{
		Tasks:               taskRepo,
		AuthClient:          authclient.NewAuthClient(cfg.AuthServiceURL),
		DeviceClient:        deviceclient.NewDeviceClient(cfg.DeviceServiceURL),
		RemindRegisterQueue: remindQueue,
		RemindCancelQueue:   cancelRemindQueue,
		TaskQueueClient:     client,
	})
}

func NewHTTPHandlerWithRepositories(ctx context.Context, repos Repositories) (string, http.Handler, error) {
	logger := slog.Default().WithGroup("task")

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

	createTaskUseCase := apptask.NewCreateTaskHandler(repos.AuthClient, repos.DeviceClient, repos.Tasks, repos.RemindRegisterQueue)
	getTaskUseCase := apptask.NewGetTaskHandler(repos.AuthClient, repos.Tasks)
	listActiveTasksUseCase := apptask.NewListActiveTasksHandler(repos.AuthClient, repos.Tasks)
	updateTaskUseCase := apptask.NewUpdateTaskHandler(repos.AuthClient, repos.Tasks)
	deleteTaskUseCase := apptask.NewDeleteTaskHandler(repos.AuthClient, repos.Tasks, repos.RemindCancelQueue)

	taskService := tasksvc.NewService(createTaskUseCase, getTaskUseCase, listActiveTasksUseCase, updateTaskUseCase, deleteTaskUseCase)

	// Register interceptor for session token extraction
	taskPath, taskHandler := taskv1connect.NewTaskServiceHandler(
		taskService,
		connect.WithInterceptors(interceptor.AuthInterceptor()),
	)
	logger.Info("task service handler registered", slog.String("path", taskPath))

	return taskPath, taskHandler, nil
}
