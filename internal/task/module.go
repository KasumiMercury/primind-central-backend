package task

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	connect "connectrpc.com/connect"
	"github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1/taskv1connect"
	apptask "github.com/KasumiMercury/primind-central-backend/internal/task/app/task"
	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/interceptor"
	tasksvc "github.com/KasumiMercury/primind-central-backend/internal/task/infra/service"
)

type Repositories struct {
	Tasks      domaintask.TaskRepository
	AuthClient authclient.AuthClient
}

// NewHTTPHandler creates a new task service HTTP handler with default auth client.
// This is the production entry point.
func NewHTTPHandler(
	ctx context.Context,
	taskRepo domaintask.TaskRepository,
	authServiceURL string,
) (string, http.Handler, error) {
	return NewHTTPHandlerWithRepositories(ctx, Repositories{
		Tasks:      taskRepo,
		AuthClient: authclient.NewAuthClient(authServiceURL),
	})
}

// NewHTTPHandlerWithRepositories creates a new task service HTTP handler with injected dependencies.
// This is useful for testing with mock implementations.
func NewHTTPHandlerWithRepositories(ctx context.Context, repos Repositories) (string, http.Handler, error) {
	logger := slog.Default().WithGroup("task")

	logger.Debug("initializing task module")

	if repos.Tasks == nil {
		return "", nil, fmt.Errorf("task repository is not configured")
	}

	if repos.AuthClient == nil {
		return "", nil, fmt.Errorf("auth client is not configured")
	}

	createTaskUseCase := apptask.NewCreateTaskHandler(repos.AuthClient, repos.Tasks)
	getTaskUseCase := apptask.NewGetTaskHandler(repos.AuthClient, repos.Tasks)
	listActiveTasksUseCase := apptask.NewListActiveTasksHandler(repos.AuthClient, repos.Tasks)

	taskService := tasksvc.NewService(createTaskUseCase, getTaskUseCase, listActiveTasksUseCase)

	// Register interceptor for session token extraction
	taskPath, taskHandler := taskv1connect.NewTaskServiceHandler(
		taskService,
		connect.WithInterceptors(interceptor.AuthInterceptor()),
	)
	logger.Info("task service handler registered", slog.String("path", taskPath))

	return taskPath, taskHandler, nil
}
