package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/period"
	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/deviceclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindcancel"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindregister"
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
	authClient        authclient.AuthClient
	deviceClient      deviceclient.DeviceClient
	taskRepo          domaintask.TaskRepository
	periodSettingRepo period.PeriodSettingRepository
	remindQueue       remindregister.Queue
	logger            *slog.Logger
}

func NewCreateTaskHandler(
	authClient authclient.AuthClient,
	deviceClient deviceclient.DeviceClient,
	taskRepo domaintask.TaskRepository,
	periodSettingRepo period.PeriodSettingRepository,
	remindQueue remindregister.Queue,
) CreateTaskUseCase {
	return &createTaskHandler{
		authClient:        authClient,
		deviceClient:      deviceClient,
		taskRepo:          taskRepo,
		periodSettingRepo: periodSettingRepo,
		remindQueue:       remindQueue,
		logger:            slog.Default().With(slog.String("module", "task")).WithGroup("task").WithGroup("createtask"),
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

	// Fetch user period settings for custom period
	var customPeriod *time.Duration

	if req.TaskType != domaintask.TypeScheduled && h.periodSettingRepo != nil {
		periodSettings, err := h.periodSettingRepo.GetByUserID(ctx, userID)
		if err != nil {
			h.logger.Warn("failed to get period settings, using defaults", slog.String("error", err.Error()))
			// Continue with default period on error
		} else if period, ok := periodSettings.GetPeriod(req.TaskType); ok {
			customPeriod = &period
			h.logger.Debug("using custom period for task type",
				slog.String("task_type", string(req.TaskType)),
				slog.Duration("period", period))
		}
	}

	task, err := domaintask.CreateTask(
		taskID,
		userID,
		req.Title,
		req.TaskType,
		req.Description,
		req.ScheduledAt,
		color,
		customPeriod,
	)
	if err != nil {
		h.logger.Warn("failed to create task entity", slog.String("error", err.Error()))

		return nil, err
	}

	devices, err := h.deviceClient.GetUserDevicesWithRetry(ctx, req.SessionToken, deviceclient.DefaultRetryConfig())
	if err != nil {
		if errors.Is(err, deviceclient.ErrUnauthorized) {
			h.logger.Info("device service: unauthorized", slog.String("error", err.Error()))

			return nil, ErrUnauthorized
		}

		if errors.Is(err, deviceclient.ErrInvalidArgument) {
			h.logger.Error("device service: invalid argument", slog.String("error", err.Error()))

			return nil, ErrDeviceInvalidArgument
		}

		h.logger.Warn("device fetch failed after retries, returning create failure",
			slog.String("task_id", task.ID().String()),
			slog.String("error", err.Error()))

		return nil, ErrDeviceServiceUnavailable
	}

	domainDevices := make([]domaintask.DeviceInfo, 0, len(devices))
	for _, d := range devices {
		domainDevices = append(domainDevices, domaintask.DeviceInfo{
			DeviceID: d.DeviceID,
			FCMToken: d.FCMToken,
		})
	}

	validDevices, filteredCount := filterDevicesWithFCMToken(domainDevices)
	if filteredCount > 0 {
		h.logger.Warn("devices filtered out due to missing FCM token",
			slog.Int("filtered_count", filteredCount),
			slog.Int("remaining_count", len(validDevices)),
			slog.Int("total_count", len(domainDevices)),
		)
	}

	var reminderInfo *domaintask.ReminderInfo
	if len(validDevices) > 0 {
		reminderInfo = domaintask.CalculateReminderTimes(task, userIDstr, validDevices)
	} else if len(domainDevices) > 0 {
		h.logger.Info("reminder registration skipped: all devices don't have FCM tokens",
			slog.String("task_id", task.ID().String()),
			slog.Int("device_count", len(domainDevices)),
		)
	}

	var remindReq *remindregister.CreateRemindRequest

	if reminderInfo != nil {
		h.logReminderInfo(reminderInfo)

		remindReq = h.convertToRemindRequest(reminderInfo)
	}

	if err := h.taskRepo.SaveTask(ctx, task); err != nil {
		h.logger.Error("failed to save task", slog.String("error", err.Error()))

		return nil, err
	}

	if remindReq != nil {
		if _, err := h.remindQueue.RegisterRemind(ctx, remindReq); err != nil {
			h.logger.Error("failed to register remind to queue",
				slog.String("task_id", task.ID().String()),
				slog.String("error", err.Error()))

			if deleteErr := h.taskRepo.DeleteTask(ctx, task.ID(), userID); deleteErr != nil {
				h.logger.Error("failed to rollback task after queue registration failure",
					slog.String("task_id", task.ID().String()),
					slog.String("error", deleteErr.Error()),
				)
			}

			return nil, ErrRemindQueueRegistrationFailed
		}
	}

	if err := h.taskRepo.UpdateTaskStatus(ctx, task.ID(), userID, domaintask.StatusActive); err != nil {
		h.logger.Error("failed to update task status to active",
			slog.String("task_id", task.ID().String()),
			slog.String("error", err.Error()))

		return nil, err
	}

	h.logger.Info("task created", slog.String("task_id", task.ID().String()))

	return &CreateTaskResult{
		TaskID:      task.ID().String(),
		Title:       task.Title(),
		TaskType:    string(task.TaskType()),
		TaskStatus:  string(domaintask.StatusActive),
		Description: task.Description(),
		ScheduledAt: task.ScheduledAt(),
		CreatedAt:   task.CreatedAt(),
		TargetAt:    task.TargetAt(),
		Color:       task.Color().String(),
	}, nil
}

func (h *createTaskHandler) logReminderInfo(info *domaintask.ReminderInfo) {
	reminderTimesStr := make([]string, len(info.ReminderTimes))
	for i, t := range info.ReminderTimes {
		reminderTimesStr[i] = t.Format(time.RFC3339)
	}

	h.logger.Info("reminder schedule prepared",
		slog.String("task_id", info.TaskID.String()),
		slog.String("task_type", string(info.TaskType)),
		slog.Int("reminder_count", len(info.ReminderTimes)),
		slog.Int("device_count", len(info.Devices)),
	)

	h.logger.Debug("reminder times calculated",
		slog.String("task_id", info.TaskID.String()),
		slog.Any("reminder_times", reminderTimesStr),
	)
}

func (h *createTaskHandler) convertToRemindRequest(info *domaintask.ReminderInfo) *remindregister.CreateRemindRequest {
	devices := make([]remindregister.DeviceRequest, 0, len(info.Devices))
	for _, d := range info.Devices {
		fcmToken := ""
		if d.FCMToken != nil {
			fcmToken = *d.FCMToken
		}

		devices = append(devices, remindregister.DeviceRequest{
			DeviceID: d.DeviceID,
			FCMToken: fcmToken,
		})
	}

	return &remindregister.CreateRemindRequest{
		Times:    info.ReminderTimes,
		UserID:   info.UserID,
		Devices:  devices,
		TaskID:   info.TaskID.String(),
		TaskType: string(info.TaskType),
		Color:    info.Color,
	}
}

func filterDevicesWithFCMToken(devices []domaintask.DeviceInfo) ([]domaintask.DeviceInfo, int) {
	valid := make([]domaintask.DeviceInfo, 0, len(devices))
	for _, d := range devices {
		if d.FCMToken != nil && *d.FCMToken != "" {
			valid = append(valid, d)
		}
	}

	filteredCount := len(devices) - len(valid)

	return valid, filteredCount
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
		logger:     slog.Default().With(slog.String("module", "task")).WithGroup("task").WithGroup("gettask"),
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
		logger:     slog.Default().With(slog.String("module", "task")).WithGroup("task").WithGroup("listactivetasks"),
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
	authClient        authclient.AuthClient
	taskRepo          domaintask.TaskRepository
	archiveRepo       domaintask.TaskArchiveRepository
	cancelRemindQueue remindcancel.Queue
	logger            *slog.Logger
}

func NewUpdateTaskHandler(
	authClient authclient.AuthClient,
	taskRepo domaintask.TaskRepository,
	archiveRepo domaintask.TaskArchiveRepository,
	cancelRemindQueue remindcancel.Queue,
) UpdateTaskUseCase {
	return &updateTaskHandler{
		authClient:        authClient,
		taskRepo:          taskRepo,
		cancelRemindQueue: cancelRemindQueue,
		archiveRepo:       archiveRepo,
		logger:            slog.Default().With(slog.String("module", "task")).WithGroup("task").WithGroup("updatetask"),
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

	if updatedTask.TaskStatus() == domaintask.StatusCompleted {
		cancelReq := &remindcancel.CancelRemindRequest{
			TaskID: req.TaskID,
			UserID: userIDstr,
		}

		if _, err := h.cancelRemindQueue.CancelRemind(ctx, cancelReq); err != nil {
			h.logger.Error("failed to cancel remind",
				slog.String("task_id", req.TaskID),
				slog.String("error", err.Error()),
			)

			return nil, ErrCancelRemindFailed
		}

		completedTask, err := domaintask.NewCompletedTask(updatedTask, time.Now())
		if err != nil {
			h.logger.Error("failed to create completed task", slog.String("error", err.Error()))

			return nil, err
		}

		if err := h.archiveRepo.ArchiveTask(ctx, completedTask, taskID, userID); err != nil {
			h.logger.Error("failed to archive task", slog.String("error", err.Error()))

			return nil, err
		}

		h.logger.Info("task completed and archived", slog.String("task_id", updatedTask.ID().String()))
	} else {
		if err := h.taskRepo.UpdateTask(ctx, updatedTask); err != nil {
			h.logger.Error("failed to update task", slog.String("error", err.Error()))

			return nil, err
		}

		h.logger.Info("task updated successfully", slog.String("task_id", updatedTask.ID().String()))
	}

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
	authClient        authclient.AuthClient
	taskRepo          domaintask.TaskRepository
	cancelRemindQueue remindcancel.Queue
	logger            *slog.Logger
}

func NewDeleteTaskHandler(
	authClient authclient.AuthClient,
	taskRepo domaintask.TaskRepository,
	cancelRemindQueue remindcancel.Queue,
) DeleteTaskUseCase {
	return &deleteTaskHandler{
		authClient:        authClient,
		taskRepo:          taskRepo,
		cancelRemindQueue: cancelRemindQueue,
		logger:            slog.Default().With(slog.String("module", "task")).WithGroup("task").WithGroup("deletetask"),
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

	// Cancel remind before deleting task
	cancelReq := &remindcancel.CancelRemindRequest{
		TaskID: req.TaskID,
		UserID: userIDstr,
	}

	if _, err := h.cancelRemindQueue.CancelRemind(ctx, cancelReq); err != nil {
		h.logger.Error("failed to cancel remind",
			slog.String("task_id", req.TaskID),
			slog.String("error", err.Error()),
		)

		return ErrCancelRemindFailed
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
