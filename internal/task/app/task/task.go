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
	TaskID       string
	SessionToken string
	Title        string
	TaskType     domaintask.Type
	Description  string
	ScheduledAt  *time.Time
	Color        string
}

type CreateTaskResult struct {
	TaskID      string
	Title       string
	TaskType    string
	TaskStatus  string
	Description string
	ScheduledAt *time.Time
	CreatedAt   time.Time
	TargetAt    time.Time
	Color       string
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

		return nil, err
	}

	var taskID *domaintask.ID

	if req.TaskID != "" {
		tid, err := domaintask.NewIDFromString(req.TaskID)
		if err != nil {
			h.logger.Warn("invalid task ID format", slog.String("error", err.Error()))

			return nil, err
		}

		taskID = &tid
	}

	if taskID != nil {
		exists, err := h.taskRepo.ExistsTaskByID(ctx, *taskID)
		if err != nil {
			h.logger.Error("failed to check task ID existence", slog.String("error", err.Error()))

			return nil, err
		}

		if exists {
			h.logger.Warn("attempted to create task with duplicate ID", slog.String("task_id", taskID.String()))

			return nil, domaintask.ErrTaskIDAlreadyExists
		}
	}

	color, err := domaintask.NewColor(req.Color)
	if err != nil {
		h.logger.Warn("invalid color format", slog.String("error", err.Error()))

		return nil, err
	}

	task, err := domaintask.CreateTask(
		taskID,
		userID,
		req.Title,
		req.TaskType,
		req.Description,
		req.ScheduledAt,
		color,
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
		TaskID:      task.ID().String(),
		Title:       task.Title(),
		TaskType:    string(task.TaskType()),
		TaskStatus:  string(task.TaskStatus()),
		Description: task.Description(),
		ScheduledAt: task.ScheduledAt(),
		CreatedAt:   task.CreatedAt(),
		TargetAt:    task.TargetAt(),
		Color:       task.Color().String(),
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
	Description string
	ScheduledAt *time.Time
	CreatedAt   time.Time
	TargetAt    time.Time
	Color       string
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
		ScheduledAt: task.ScheduledAt(),
		CreatedAt:   task.CreatedAt(),
		TargetAt:    task.TargetAt(),
		Color:       task.Color().String(),
	}, nil
}
