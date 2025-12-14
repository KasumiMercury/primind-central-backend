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

type ListActiveTasksRequest struct {
	SessionToken string
	SortType     domaintask.SortType
}

type ListActiveTasksResult struct {
	Tasks []TaskItem
}

type TaskItem struct {
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

type ListActiveTasksUseCase interface {
	ListActiveTasks(ctx context.Context, req *ListActiveTasksRequest) (*ListActiveTasksResult, error)
}

type listActiveTasksHandler struct {
	authClient authclient.AuthClient
	taskRepo   domaintask.TaskRepository
	logger     *slog.Logger
}

func NewListActiveTasksHandler(
	authClient authclient.AuthClient,
	taskRepo domaintask.TaskRepository,
) ListActiveTasksUseCase {
	return &listActiveTasksHandler{
		authClient: authClient,
		taskRepo:   taskRepo,
		logger:     slog.Default().WithGroup("task").WithGroup("listactivetasks"),
	}
}

func (h *listActiveTasksHandler) ListActiveTasks(ctx context.Context, req *ListActiveTasksRequest) (*ListActiveTasksResult, error) {
	if req == nil {
		return nil, ErrListActiveTasksRequestRequired
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

	tasks, err := h.taskRepo.ListActiveTasksByUserID(ctx, userID, req.SortType)
	if err != nil {
		h.logger.Error("failed to list active tasks", slog.String("error", err.Error()))

		return nil, err
	}

	result := &ListActiveTasksResult{
		Tasks: make([]TaskItem, 0, len(tasks)),
	}

	for _, task := range tasks {
		result.Tasks = append(result.Tasks, TaskItem{
			TaskID:      task.ID().String(),
			Title:       task.Title(),
			TaskType:    task.TaskType(),
			TaskStatus:  task.TaskStatus(),
			Description: task.Description(),
			ScheduledAt: task.ScheduledAt(),
			CreatedAt:   task.CreatedAt(),
			TargetAt:    task.TargetAt(),
			Color:       task.Color().String(),
		})
	}

	h.logger.Info("active tasks listed", slog.Int("count", len(result.Tasks)))

	return result, nil
}

type UpdateTaskRequest struct {
	SessionToken     string
	TaskID           string
	UpdateMask       []string
	TaskStatus       *domaintask.Status
	Title            *string
	Description      *string
	ScheduledAt      *time.Time
	ClearScheduledAt bool
	Color            *string
}

type UpdateTaskResult struct {
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

type UpdateTaskUseCase interface {
	UpdateTask(ctx context.Context, req *UpdateTaskRequest) (*UpdateTaskResult, error)
}

type updateTaskHandler struct {
	authClient authclient.AuthClient
	taskRepo   domaintask.TaskRepository
	logger     *slog.Logger
}

func NewUpdateTaskHandler(
	authClient authclient.AuthClient,
	taskRepo domaintask.TaskRepository,
) UpdateTaskUseCase {
	return &updateTaskHandler{
		authClient: authClient,
		taskRepo:   taskRepo,
		logger:     slog.Default().WithGroup("task").WithGroup("updatetask"),
	}
}

func (h *updateTaskHandler) UpdateTask(ctx context.Context, req *UpdateTaskRequest) (*UpdateTaskResult, error) {
	if req == nil {
		return nil, ErrUpdateTaskRequestRequired
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

	if req.TaskID == "" {
		return nil, ErrTaskIDRequired
	}

	taskID, err := domaintask.NewIDFromString(req.TaskID)
	if err != nil {
		h.logger.Warn("invalid task ID format", slog.String("error", err.Error()))

		return nil, err
	}

	if len(req.UpdateMask) == 0 {
		return nil, domaintask.ErrNoFieldsToUpdate
	}

	updateInput, err := h.buildUpdateInput(req)
	if err != nil {
		return nil, err
	}

	existingTask, err := h.taskRepo.GetTaskByID(ctx, taskID, userID)
	if err != nil {
		if errors.Is(err, domaintask.ErrTaskNotFound) {
			h.logger.Info("task not found", slog.String("task_id", req.TaskID))

			return nil, ErrTaskNotFound
		}

		h.logger.Error("failed to get task", slog.String("error", err.Error()))

		return nil, err
	}

	if err := h.validateScheduledAtConstraint(existingTask, updateInput); err != nil {
		h.logger.Warn("scheduled_at validation failed", slog.String("error", err.Error()))

		return nil, err
	}

	updatedTask, err := existingTask.ApplyUpdate(updateInput)
	if err != nil {
		h.logger.Warn("failed to apply update", slog.String("error", err.Error()))

		return nil, err
	}

	if err := h.taskRepo.UpdateTask(ctx, updatedTask); err != nil {
		h.logger.Error("failed to update task", slog.String("error", err.Error()))

		return nil, err
	}

	h.logger.Info("task updated successfully", slog.String("task_id", updatedTask.ID().String()))

	return &UpdateTaskResult{
		TaskID:      updatedTask.ID().String(),
		Title:       updatedTask.Title(),
		TaskType:    updatedTask.TaskType(),
		TaskStatus:  updatedTask.TaskStatus(),
		Description: updatedTask.Description(),
		ScheduledAt: updatedTask.ScheduledAt(),
		CreatedAt:   updatedTask.CreatedAt(),
		TargetAt:    updatedTask.TargetAt(),
		Color:       updatedTask.Color().String(),
	}, nil
}

func (h *updateTaskHandler) buildUpdateInput(req *UpdateTaskRequest) (*domaintask.TaskUpdateInput, error) {
	input := &domaintask.TaskUpdateInput{}

	validFields := map[string]bool{
		"task_status":  true,
		"title":        true,
		"description":  true,
		"scheduled_at": true,
		"color":        true,
	}

	for _, field := range req.UpdateMask {
		if !validFields[field] {
			return nil, fmt.Errorf("%w: %s", domaintask.ErrInvalidUpdateField, field)
		}

		switch field {
		case "task_status":
			if req.TaskStatus != nil {
				input.TaskStatus = req.TaskStatus
			}
		case "title":
			if req.Title != nil {
				input.Title = req.Title
			}
		case "description":
			if req.Description != nil {
				input.Description = req.Description
			}
		case "scheduled_at":
			if req.ClearScheduledAt {
				input.ClearScheduledAt = true
			} else if req.ScheduledAt != nil {
				input.ScheduledAt = req.ScheduledAt
			}
		case "color":
			if req.Color != nil {
				color, err := domaintask.NewColor(*req.Color)
				if err != nil {
					return nil, err
				}

				input.Color = &color
			}
		}
	}

	return input, nil
}

func (h *updateTaskHandler) validateScheduledAtConstraint(task *domaintask.Task, input *domaintask.TaskUpdateInput) error {
	if input.ScheduledAt == nil && !input.ClearScheduledAt {
		return nil
	}

	if task.TaskType() != domaintask.TypeScheduled {
		return domaintask.ErrScheduledAtNotAllowed
	}

	return nil
}

type DeleteTaskRequest struct {
	SessionToken string
	TaskID       string
}

type DeleteTaskUseCase interface {
	DeleteTask(ctx context.Context, req *DeleteTaskRequest) error
}

type deleteTaskHandler struct {
	authClient authclient.AuthClient
	taskRepo   domaintask.TaskRepository
	logger     *slog.Logger
}

func NewDeleteTaskHandler(
	authClient authclient.AuthClient,
	taskRepo domaintask.TaskRepository,
) DeleteTaskUseCase {
	return &deleteTaskHandler{
		authClient: authClient,
		taskRepo:   taskRepo,
		logger:     slog.Default().WithGroup("task").WithGroup("deletetask"),
	}
}

func (h *deleteTaskHandler) DeleteTask(ctx context.Context, req *DeleteTaskRequest) error {
	if req == nil {
		return ErrDeleteTaskRequestRequired
	}

	userIDstr, err := h.authClient.ValidateSession(ctx, req.SessionToken)
	if err != nil {
		if errors.Is(err, authclient.ErrUnauthorized) {
			h.logger.Info("session validation failed", slog.String("error", err.Error()))

			return ErrUnauthorized
		}

		h.logger.Error("session validation failed", slog.String("error", err.Error()))

		return fmt.Errorf("session validation failed: %w", err)
	}

	userID, err := domainuser.NewIDFromString(userIDstr)
	if err != nil {
		h.logger.Warn("invalid user ID format", slog.String("error", err.Error()))

		return err
	}

	if req.TaskID == "" {
		h.logger.Warn("delete task called with empty task ID")

		return ErrTaskIDRequired
	}

	taskID, err := domaintask.NewIDFromString(req.TaskID)
	if err != nil {
		h.logger.Warn("invalid task ID format", slog.String("error", err.Error()))

		return err
	}

	if err := h.taskRepo.DeleteTask(ctx, taskID, userID); err != nil {
		if errors.Is(err, domaintask.ErrTaskNotFound) {
			h.logger.Info("task not found", slog.String("task_id", req.TaskID))

			return ErrTaskNotFound
		}

		h.logger.Error("failed to delete task", slog.String("error", err.Error()))

		return err
	}

	h.logger.Info("task deleted successfully", slog.String("task_id", req.TaskID))

	return nil
}
