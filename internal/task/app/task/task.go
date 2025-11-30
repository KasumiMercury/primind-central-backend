package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
)

type CreateTaskRequest struct {
	SessionToken string
	Title        string
	TaskType     domaintask.Type
	Description  *string
	DueTime      *time.Time
}

type CreateTaskResult struct {
	TaskID string
}

type CreateTaskUseCase interface {
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResult, error)
}

type createTaskHandler struct {
	authClient authclient.AuthClient
	taskRepo   domaintask.TaskRepository
	logger     *slog.Logger
}

func NewCreateTaskHandler(
	authClient authclient.AuthClient,
	taskRepo domaintask.TaskRepository,
) CreateTaskUseCase {
	return &createTaskHandler{
		authClient: authClient,
		taskRepo:   taskRepo,
		logger:     slog.Default().WithGroup("task").WithGroup("createtask"),
	}
}

func (h *createTaskHandler) CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResult, error) {
	if req == nil {
		return nil, ErrCreateTaskRequestRequired
	}

	userIDstr, err := h.authClient.ValidateSession(ctx, req.SessionToken)
	if err != nil {
		h.logger.Info("session validation failed", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %s", ErrUnauthorized, err.Error())
	}

	userID, err := domainuser.NewIDFromString(userIDstr)
	if err != nil {
		h.logger.Warn("invalid user ID format", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %s", ErrUnauthorized, err.Error())
	}

	if req.Title == "" {
		h.logger.Warn("create task called with empty title")

		return nil, ErrTitleRequired
	}

	task, err := domaintask.CreateTask(
		userID,
		req.Title,
		req.TaskType,
		req.Description,
		req.DueTime,
	)
	if err != nil {
		h.logger.Warn("failed to create task entity", slog.String("error", err.Error()))

		return nil, err
	}

	if err := h.taskRepo.SaveTask(ctx, task); err != nil {
		h.logger.Error("failed to save task", slog.String("error", err.Error()))

		return nil, err
	}

	h.logger.Info("task created successfully", slog.String("task_id", task.ID().String()))

	return &CreateTaskResult{
		TaskID: task.ID().String(),
	}, nil
}

type GetTaskRequest struct {
	SessionToken string
	TaskID       string
}

type GetTaskResult struct {
	TaskID      string
	Title       string
	TaskType    domaintask.Type
	TaskStatus  domaintask.Status
	Description *string
	DueTime     *time.Time
	CreatedAt   time.Time
}

type GetTaskUseCase interface {
	GetTask(ctx context.Context, req *GetTaskRequest) (*GetTaskResult, error)
}

type getTaskHandler struct {
	authClient authclient.AuthClient
	taskRepo   domaintask.TaskRepository
	logger     *slog.Logger
}

func NewGetTaskHandler(
	authClient authclient.AuthClient,
	taskRepo domaintask.TaskRepository,
) GetTaskUseCase {
	return &getTaskHandler{
		authClient: authClient,
		taskRepo:   taskRepo,
		logger:     slog.Default().WithGroup("task").WithGroup("gettask"),
	}
}

func (h *getTaskHandler) GetTask(ctx context.Context, req *GetTaskRequest) (*GetTaskResult, error) {
	if req == nil {
		return nil, ErrGetTaskRequestRequired
	}

	userIDstr, err := h.authClient.ValidateSession(ctx, req.SessionToken)
	if err != nil {
		h.logger.Info("session validation failed", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %s", ErrUnauthorized, err.Error())
	}

	userID, err := domainuser.NewIDFromString(userIDstr)
	if err != nil {
		h.logger.Warn("invalid user ID format", slog.String("error", err.Error()))

		return nil, err
	}

	if req.TaskID == "" {
		h.logger.Warn("get task called with empty task ID")

		return nil, ErrTaskIDRequired
	}

	taskID, err := domaintask.NewIDFromString(req.TaskID)
	if err != nil {
		h.logger.Warn("invalid task ID format", slog.String("error", err.Error()))

		return nil, err
	}

	task, err := h.taskRepo.GetTaskByID(ctx, taskID, userID)
	if err != nil {
		if errors.Is(err, domaintask.ErrTaskNotFound) {
			h.logger.Info("task not found", slog.String("task_id", req.TaskID))

			return nil, ErrTaskNotFound
		}

		h.logger.Error("failed to get task", slog.String("error", err.Error()))

		return nil, err
	}

	return &GetTaskResult{
		TaskID:      task.ID().String(),
		Title:       task.Title(),
		TaskType:    task.TaskType(),
		TaskStatus:  task.TaskStatus(),
		Description: task.Description(),
		DueTime:     task.DueTime(),
		CreatedAt:   task.CreatedAt(),
	}, nil
}
