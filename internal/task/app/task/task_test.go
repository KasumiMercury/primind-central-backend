package task

import (
	"context"
	"errors"
	"testing"
	"time"

	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	domainuser "github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/testutil"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestCreateTaskSuccess(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	tests := []struct {
		name   string
		req    CreateTaskRequest
		userID domainuser.ID
	}{
		{
			name: "create normal task without scheduled time",
			req: CreateTaskRequest{
				SessionToken: "token-normal",
				Title:        "Test Task",
				TaskType:     domaintask.TypeNormal,
				Color:        "#FF6B6B",
			},
			userID: userID,
		},
		{
			name: "create task with scheduled time and description",
			req: func() CreateTaskRequest {
				desc := "task description"
				scheduled := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)

				return CreateTaskRequest{
					SessionToken: "token-scheduled",
					Title:        "Task with scheduled time",
					TaskType:     domaintask.TypeScheduled,
					Description:  desc,
					ScheduledAt:  &scheduled,
					Color:        "#4ECDC4",
				}
			}(),
			userID: userID,
		},
		{
			name: "create task with predefined valid task ID",
			req: func() CreateTaskRequest {
				validTaskID, err := domaintask.NewID()
				if err != nil {
					t.Fatalf("failed to generate task ID: %v", err)
				}

				return CreateTaskRequest{
					TaskID:       validTaskID.String(),
					SessionToken: "token-with-id",
					Title:        "Task with predefined ID",
					TaskType:     domaintask.TypeNormal,
					Description:  "This task has a predefined ID",
					Color:        "#FFD166",
				}
			}(),
			userID: userID,
		},
		{
			name: "empty title",
			req: func() CreateTaskRequest {
				validTaskID, err := domaintask.NewID()
				if err != nil {
					t.Fatalf("failed to generate task ID: %v", err)
				}

				return CreateTaskRequest{
					TaskID:       validTaskID.String(),
					SessionToken: "token-empty-title",
					Title:        "",
					TaskType:     domaintask.TypeNormal,
					Description:  "This task has an empty title",
					Color:        "#5E60CE",
				}
			}(),
			userID: userID,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAuth := NewMockAuthClient(ctrl)
			mockAuth.EXPECT().ValidateSession(gomock.Any(), tt.req.SessionToken).Return(tt.userID.String(), nil)

			handler := NewCreateTaskHandler(mockAuth, repo)

			resp, err := handler.CreateTask(ctx, &tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp == nil || resp.TaskID == "" {
				t.Fatalf("expected task id, got %#v", resp)
			}

			taskID, err := domaintask.NewIDFromString(resp.TaskID)
			if err != nil {
				t.Fatalf("invalid task id returned: %v", err)
			}

			saved, err := repo.GetTaskByID(ctx, taskID, tt.userID)
			if err != nil {
				t.Fatalf("failed to fetch saved task: %v", err)
			}

			if saved.Title() != tt.req.Title {
				t.Fatalf("expected title %q, got %q", tt.req.Title, saved.Title())
			}

			if saved.TaskType() != tt.req.TaskType {
				t.Fatalf("expected task type %q, got %q", tt.req.TaskType, saved.TaskType())
			}

			if saved.TaskStatus() != domaintask.StatusActive {
				t.Fatalf("expected task status %q, got %q", domaintask.StatusActive, saved.TaskStatus())
			}

			if saved.Description() != tt.req.Description {
				t.Fatalf("expected description %v, got %v", tt.req.Description, saved.Description())
			}

			if tt.req.ScheduledAt == nil && saved.ScheduledAt() != nil {
				t.Fatalf("expected nil scheduled time, got %v", saved.ScheduledAt())
			}

			if tt.req.ScheduledAt != nil {
				if saved.ScheduledAt() == nil {
					t.Fatalf("expected scheduled time, got nil")
				}

				if !saved.ScheduledAt().Equal(*tt.req.ScheduledAt) {
					t.Fatalf("expected scheduled time %v, got %v", tt.req.ScheduledAt, saved.ScheduledAt())
				}
			}

			// Verify TargetAt is set in the response
			if resp.TargetAt.IsZero() {
				t.Fatalf("expected target at to be set, got zero time")
			}

			// Verify TargetAt matches saved task
			if !resp.TargetAt.Equal(saved.TargetAt()) {
				t.Fatalf("expected target at %v, got %v", saved.TargetAt(), resp.TargetAt)
			}
		})
	}
}

func TestCreateTaskError(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	validUserID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	tests := []struct {
		name        string
		req         *CreateTaskRequest
		setupAuth   func(ctrl *gomock.Controller) authclient.AuthClient
		expectedErr error
	}{
		{
			name:        "nil request",
			req:         nil,
			setupAuth:   func(ctrl *gomock.Controller) authclient.AuthClient { return NewMockAuthClient(ctrl) },
			expectedErr: ErrCreateTaskRequestRequired,
		},
		{
			name: "unauthorized session",
			req: &CreateTaskRequest{
				SessionToken: "invalid-token",
				Title:        "title",
				TaskType:     domaintask.TypeNormal,
				Color:        "#FF6B6B",
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "invalid-token").
					Return("", authclient.ErrUnauthorized)

				return mockAuth
			},
			expectedErr: ErrUnauthorized,
		},
		{
			name: "scheduled time required for has_scheduled_time",
			req: &CreateTaskRequest{
				SessionToken: "token",
				Title:        "task without scheduled time",
				TaskType:     domaintask.TypeScheduled,
				Color:        "#FF6B6B",
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrScheduledAtRequired,
		},
		{
			name: "invalid task ID format",
			req: &CreateTaskRequest{
				TaskID:       "not-a-uuid",
				SessionToken: "token",
				Title:        "Task with invalid ID",
				TaskType:     domaintask.TypeNormal,
				Color:        "#FF6B6B",
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrIDInvalidFormat,
		},
		{
			name: "task ID is UUIDv4 not v7",
			req: func() *CreateTaskRequest {
				uuidv4 := uuid.New()

				return &CreateTaskRequest{
					TaskID:       uuidv4.String(),
					SessionToken: "token",
					Title:        "Task with UUIDv4",
					TaskType:     domaintask.TypeNormal,
					Color:        "#FF6B6B",
				}
			}(),
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrIDInvalidV7,
		},
		{
			name: "duplicate task ID from same user",
			req: func() *CreateTaskRequest {
				existingTaskID, _ := domaintask.NewID()
				color := domaintask.MustColor("#FF6B6B")
				existingTask, _ := domaintask.CreateTask(
					&existingTaskID,
					validUserID,
					"Existing Task",
					domaintask.TypeNormal,
					"",
					nil,
					color,
				)
				_ = repo.SaveTask(context.Background(), existingTask)

				return &CreateTaskRequest{
					TaskID:       existingTaskID.String(),
					SessionToken: "token",
					Title:        "Duplicate Task",
					TaskType:     domaintask.TypeNormal,
					Color:        "#FF6B6B",
				}
			}(),
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrTaskIDAlreadyExists,
		},
		{
			name: "duplicate task ID from different user",
			req: func() *CreateTaskRequest {
				user1ID, _ := domainuser.NewID()
				existingTaskID, _ := domaintask.NewID()
				color := domaintask.MustColor("#FF6B6B")
				existingTask, _ := domaintask.CreateTask(
					&existingTaskID,
					user1ID,
					"User1's Task",
					domaintask.TypeNormal,
					"",
					nil,
					color,
				)
				_ = repo.SaveTask(context.Background(), existingTask)

				return &CreateTaskRequest{
					TaskID:       existingTaskID.String(),
					SessionToken: "token",
					Title:        "User2's Task with same ID",
					TaskType:     domaintask.TypeNormal,
					Color:        "#FF6B6B",
				}
			}(),
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrTaskIDAlreadyExists,
		},
		{
			name: "empty color",
			req: &CreateTaskRequest{
				SessionToken: "token",
				Title:        "Task without color",
				TaskType:     domaintask.TypeNormal,
				Color:        "",
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrColorEmpty,
		},
		{
			name: "invalid color format",
			req: &CreateTaskRequest{
				SessionToken: "token",
				Title:        "Task with invalid color",
				TaskType:     domaintask.TypeNormal,
				Color:        "#FFF",
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrColorInvalidFormat,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAuth := tt.setupAuth(ctrl)
			handler := NewCreateTaskHandler(mockAuth, repo)

			_, err := handler.CreateTask(ctx, tt.req)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}

			if !errors.Is(err, tt.expectedErr) && err.Error() != tt.expectedErr.Error() {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestGetTaskSuccess(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	desc := "stored task"
	scheduled := time.Now().Add(3 * time.Hour).UTC().Truncate(time.Second)

	userIDNormal, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	userIDWithscheduled, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	now := time.Now().UTC()

	validColor := domaintask.MustColor("#FF6B6B")
	taskWithNoscheduled := createPersistedTask(t, repo, userIDNormal, "stored", domaintask.TypeNormal, desc, nil, now, validColor)
	taskWithscheduled := createPersistedTask(t, repo, userIDWithscheduled, "stored with scheduled", domaintask.TypeScheduled, desc, &scheduled, now, validColor)

	tests := []struct {
		name         string
		req          GetTaskRequest
		userID       domainuser.ID
		expectedTask *domaintask.Task
	}{
		{
			name: "get normal task",
			req: GetTaskRequest{
				SessionToken: "token-normal",
				TaskID:       taskWithNoscheduled.ID().String(),
			},
			userID:       userIDNormal,
			expectedTask: taskWithNoscheduled,
		},
		{
			name: "get task with scheduled time",
			req: GetTaskRequest{
				SessionToken: "token-scheduled",
				TaskID:       taskWithscheduled.ID().String(),
			},
			userID:       userIDWithscheduled,
			expectedTask: taskWithscheduled,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAuth := NewMockAuthClient(ctrl)
			mockAuth.EXPECT().ValidateSession(gomock.Any(), tt.req.SessionToken).
				Return(tt.userID.String(), nil)

			handler := NewGetTaskHandler(mockAuth, repo)

			resp, err := handler.GetTask(ctx, &tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.TaskID != tt.expectedTask.ID().String() {
				t.Fatalf("expected task id %s, got %s", tt.expectedTask.ID().String(), resp.TaskID)
			}

			if resp.Title != tt.expectedTask.Title() {
				t.Fatalf("expected title %q, got %q", tt.expectedTask.Title(), resp.Title)
			}

			if resp.TaskType != tt.expectedTask.TaskType() {
				t.Fatalf("expected task type %q, got %q", tt.expectedTask.TaskType(), resp.TaskType)
			}

			if resp.TaskStatus != tt.expectedTask.TaskStatus() {
				t.Fatalf("expected status %q, got %q", tt.expectedTask.TaskStatus(), resp.TaskStatus)
			}

			if resp.Description != tt.expectedTask.Description() {
				t.Fatalf("expected description %v, got %v", tt.expectedTask.Description(), resp.Description)
			}

			if tt.expectedTask.ScheduledAt() == nil && resp.ScheduledAt != nil {
				t.Fatalf("expected nil scheduled time, got %v", resp.ScheduledAt)
			}

			if tt.expectedTask.ScheduledAt() != nil {
				if resp.ScheduledAt == nil || !resp.ScheduledAt.Equal(*tt.expectedTask.ScheduledAt()) {
					t.Fatalf("expected scheduled time %v, got %v", tt.expectedTask.ScheduledAt(), resp.ScheduledAt)
				}
			}

			if !resp.CreatedAt.Equal(tt.expectedTask.CreatedAt()) {
				t.Fatalf("expected created at %v, got %v", tt.expectedTask.CreatedAt(), resp.CreatedAt)
			}

			if !resp.TargetAt.Equal(tt.expectedTask.TargetAt()) {
				t.Fatalf("expected target at %v, got %v", tt.expectedTask.TargetAt(), resp.TargetAt)
			}

			if resp.Color != tt.expectedTask.Color().String() {
				t.Fatalf("expected color %q, got %q", tt.expectedTask.Color().String(), resp.Color)
			}
		})
	}
}

func TestGetTaskError(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	validUserID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	missingID, err := domaintask.NewID()
	if err != nil {
		t.Fatalf("failed to generate id: %v", err)
	}

	tests := []struct {
		name        string
		req         *GetTaskRequest
		setupAuth   func(ctrl *gomock.Controller) authclient.AuthClient
		expectedErr error
	}{
		{
			name:        "nil request",
			req:         nil,
			setupAuth:   func(ctrl *gomock.Controller) authclient.AuthClient { return NewMockAuthClient(ctrl) },
			expectedErr: ErrGetTaskRequestRequired,
		},
		{
			name: "unauthorized session",
			req: &GetTaskRequest{
				SessionToken: "bad-token",
				TaskID:       missingID.String(),
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "bad-token").
					Return("", authclient.ErrUnauthorized)

				return mockAuth
			},
			expectedErr: ErrUnauthorized,
		},
		{
			name: "empty task id",
			req: &GetTaskRequest{
				SessionToken: "token",
				TaskID:       "",
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: ErrTaskIDRequired,
		},
		{
			name: "invalid task id format",
			req: &GetTaskRequest{
				SessionToken: "token",
				TaskID:       "invalid-uuid",
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrIDInvalidFormat,
		},
		{
			name: "task not found",
			req: &GetTaskRequest{
				SessionToken: "token",
				TaskID:       missingID.String(),
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: ErrTaskNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAuth := tt.setupAuth(ctrl)
			handler := NewGetTaskHandler(mockAuth, repo)

			_, err := handler.GetTask(ctx, tt.req)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}

			if !errors.Is(err, tt.expectedErr) && err.Error() != tt.expectedErr.Error() {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func setupTaskRepository(t *testing.T) domaintask.TaskRepository {
	t.Helper()

	ctx := context.Background()
	db, cleanup := testutil.SetupPostgresContainer(ctx, t)
	t.Cleanup(cleanup)

	if err := db.AutoMigrate(&repository.TaskModel{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return repository.NewTaskRepository(db)
}

func createPersistedTask(
	t *testing.T,
	repo domaintask.TaskRepository,
	userID domainuser.ID,
	title string,
	taskType domaintask.Type,
	description string,
	scheduledAt *time.Time,
	createdAt time.Time,
	color domaintask.Color,
) *domaintask.Task {
	t.Helper()

	id, err := domaintask.NewID()
	if err != nil {
		t.Fatalf("failed to generate task id: %v", err)
	}

	// Calculate targetAt based on task type (same logic as CreateTask)
	var targetAt time.Time
	if taskType == domaintask.TypeScheduled && scheduledAt != nil {
		targetAt = *scheduledAt
	} else {
		activePeriod := domaintask.GetActivePeriodForType(taskType)
		targetAt = createdAt.Add(time.Duration(activePeriod))
	}

	task, err := domaintask.NewTask(
		id,
		userID,
		title,
		taskType,
		domaintask.StatusActive,
		description,
		scheduledAt,
		createdAt,
		targetAt,
		color,
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := repo.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	return task
}

func createPersistedTaskWithStatus(
	t *testing.T,
	repo domaintask.TaskRepository,
	userID domainuser.ID,
	title string,
	taskType domaintask.Type,
	taskStatus domaintask.Status,
	createdAt time.Time,
	targetAt time.Time,
	color domaintask.Color,
) *domaintask.Task {
	t.Helper()

	id, err := domaintask.NewID()
	if err != nil {
		t.Fatalf("failed to generate task id: %v", err)
	}

	task, err := domaintask.NewTask(
		id,
		userID,
		title,
		taskType,
		taskStatus,
		"",
		nil,
		createdAt,
		targetAt,
		color,
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := repo.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	return task
}

func TestListActiveTasksSuccess(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Create tasks with different target_at and created_at times
	task1 := createPersistedTaskWithStatus(t, repo, userID, "Task 1", domaintask.TypeNormal, domaintask.StatusActive, now, now.Add(1*time.Hour), domaintask.MustColor("#FF6B6B"))
	task2 := createPersistedTaskWithStatus(t, repo, userID, "Task 2", domaintask.TypeUrgent, domaintask.StatusActive, now, now.Add(30*time.Minute), domaintask.MustColor("#4ECDC4"))
	// Task 3: same target_at as task2, but newer created_at - should come first
	task3 := createPersistedTaskWithStatus(t, repo, userID, "Task 3", domaintask.TypeNormal, domaintask.StatusActive, now.Add(1*time.Second), now.Add(30*time.Minute), domaintask.MustColor("#45B7D1"))
	// Task 4: COMPLETED status - should not be returned
	_ = createPersistedTaskWithStatus(t, repo, userID, "Task 4", domaintask.TypeNormal, domaintask.StatusCompleted, now, now.Add(20*time.Minute), domaintask.MustColor("#96CEB4"))

	ctrl := gomock.NewController(t)
	mockAuth := NewMockAuthClient(ctrl)
	mockAuth.EXPECT().ValidateSession(gomock.Any(), "valid-token").Return(userID.String(), nil)

	handler := NewListActiveTasksHandler(mockAuth, repo)

	result, err := handler.ListActiveTasks(ctx, &ListActiveTasksRequest{
		SessionToken: "valid-token",
		SortType:     domaintask.SortTypeTargetAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result.Tasks))
	}

	// Expected order: task3 (30min, newer), task2 (30min, older), task1 (1h)
	expectedOrder := []string{task3.ID().String(), task2.ID().String(), task1.ID().String()}
	for i, task := range result.Tasks {
		if task.TaskID != expectedOrder[i] {
			t.Errorf("position %d: expected %s, got %s", i, expectedOrder[i], task.TaskID)
		}
	}
}

func TestListActiveTasksEmpty(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	ctrl := gomock.NewController(t)
	mockAuth := NewMockAuthClient(ctrl)
	mockAuth.EXPECT().ValidateSession(gomock.Any(), "valid-token").Return(userID.String(), nil)

	handler := NewListActiveTasksHandler(mockAuth, repo)

	result, err := handler.ListActiveTasks(ctx, &ListActiveTasksRequest{
		SessionToken: "valid-token",
		SortType:     domaintask.SortTypeTargetAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(result.Tasks))
	}
}

func TestListActiveTasksError(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	validUserID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	tests := []struct {
		name        string
		req         *ListActiveTasksRequest
		setupAuth   func(ctrl *gomock.Controller) authclient.AuthClient
		expectedErr error
	}{
		{
			name:        "nil request",
			req:         nil,
			setupAuth:   func(ctrl *gomock.Controller) authclient.AuthClient { return NewMockAuthClient(ctrl) },
			expectedErr: ErrListActiveTasksRequestRequired,
		},
		{
			name: "unauthorized session",
			req: &ListActiveTasksRequest{
				SessionToken: "invalid-token",
				SortType:     domaintask.SortTypeTargetAt,
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "invalid-token").
					Return("", authclient.ErrUnauthorized)

				return mockAuth
			},
			expectedErr: ErrUnauthorized,
		},
		{
			name: "invalid sort type",
			req: &ListActiveTasksRequest{
				SessionToken: "valid-token",
				SortType:     domaintask.SortType("invalid"),
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "valid-token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrInvalidSortType,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAuth := tt.setupAuth(ctrl)
			handler := NewListActiveTasksHandler(mockAuth, repo)

			_, err := handler.ListActiveTasks(ctx, tt.req)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestUpdateTaskSuccess(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	userID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	scheduled := now.Add(2 * time.Hour)
	validColor := domaintask.MustColor("#FF6B6B")

	// Create a normal task
	normalTask := createPersistedTask(t, repo, userID, "Original Title", domaintask.TypeNormal, "Original Description", nil, now, validColor)

	// Create a scheduled task
	scheduledTask := createPersistedTask(t, repo, userID, "Scheduled Task", domaintask.TypeScheduled, "Scheduled Description", &scheduled, now, validColor)

	tests := []struct {
		name          string
		taskID        string
		req           UpdateTaskRequest
		expectedTitle string
		expectedDesc  string
	}{
		{
			name:   "update task_status only",
			taskID: normalTask.ID().String(),
			req: func() UpdateTaskRequest {
				status := domaintask.StatusCompleted
				return UpdateTaskRequest{
					SessionToken: "token",
					TaskID:       normalTask.ID().String(),
					UpdateMask:   []string{"task_status"},
					TaskStatus:   &status,
				}
			}(),
			expectedTitle: "Original Title",
			expectedDesc:  "Original Description",
		},
		{
			name:   "update title only",
			taskID: normalTask.ID().String(),
			req: func() UpdateTaskRequest {
				title := "Updated Title"
				return UpdateTaskRequest{
					SessionToken: "token",
					TaskID:       normalTask.ID().String(),
					UpdateMask:   []string{"title"},
					Title:        &title,
				}
			}(),
			expectedTitle: "Updated Title",
			expectedDesc:  "Original Description",
		},
		{
			name:   "update description only",
			taskID: normalTask.ID().String(),
			req: func() UpdateTaskRequest {
				desc := "Updated Description"
				return UpdateTaskRequest{
					SessionToken: "token",
					TaskID:       normalTask.ID().String(),
					UpdateMask:   []string{"description"},
					Description:  &desc,
				}
			}(),
			expectedTitle: "Updated Title",
			expectedDesc:  "Updated Description",
		},
		{
			name:   "update color only",
			taskID: normalTask.ID().String(),
			req: func() UpdateTaskRequest {
				color := "#4ECDC4"
				return UpdateTaskRequest{
					SessionToken: "token",
					TaskID:       normalTask.ID().String(),
					UpdateMask:   []string{"color"},
					Color:        &color,
				}
			}(),
			expectedTitle: "Updated Title",
			expectedDesc:  "Updated Description",
		},
		{
			name:   "update scheduled_at for SCHEDULED task",
			taskID: scheduledTask.ID().String(),
			req: func() UpdateTaskRequest {
				newScheduled := now.Add(4 * time.Hour)
				return UpdateTaskRequest{
					SessionToken: "token",
					TaskID:       scheduledTask.ID().String(),
					UpdateMask:   []string{"scheduled_at"},
					ScheduledAt:  &newScheduled,
				}
			}(),
			expectedTitle: "Scheduled Task",
			expectedDesc:  "Scheduled Description",
		},
		{
			name:   "update multiple fields",
			taskID: normalTask.ID().String(),
			req: func() UpdateTaskRequest {
				title := "Multi Update Title"
				desc := "Multi Update Desc"
				color := "#45B7D1"
				return UpdateTaskRequest{
					SessionToken: "token",
					TaskID:       normalTask.ID().String(),
					UpdateMask:   []string{"title", "description", "color"},
					Title:        &title,
					Description:  &desc,
					Color:        &color,
				}
			}(),
			expectedTitle: "Multi Update Title",
			expectedDesc:  "Multi Update Desc",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAuth := NewMockAuthClient(ctrl)
			mockAuth.EXPECT().ValidateSession(gomock.Any(), tt.req.SessionToken).
				Return(userID.String(), nil)

			handler := NewUpdateTaskHandler(mockAuth, repo)

			resp, err := handler.UpdateTask(ctx, &tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatalf("expected response, got nil")
			}

			if resp.TaskID != tt.taskID {
				t.Fatalf("expected task id %s, got %s", tt.taskID, resp.TaskID)
			}

			if resp.Title != tt.expectedTitle {
				t.Fatalf("expected title %q, got %q", tt.expectedTitle, resp.Title)
			}

			if resp.Description != tt.expectedDesc {
				t.Fatalf("expected description %q, got %q", tt.expectedDesc, resp.Description)
			}
		})
	}
}

func TestUpdateTaskError(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	validUserID, err := domainuser.NewID()
	if err != nil {
		t.Fatalf("failed to generate user id: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	validColor := domaintask.MustColor("#FF6B6B")

	// Create a normal task for testing
	normalTask := createPersistedTask(t, repo, validUserID, "Test Task", domaintask.TypeNormal, "", nil, now, validColor)

	missingID, err := domaintask.NewID()
	if err != nil {
		t.Fatalf("failed to generate id: %v", err)
	}

	tests := []struct {
		name        string
		req         *UpdateTaskRequest
		setupAuth   func(ctrl *gomock.Controller) authclient.AuthClient
		expectedErr error
	}{
		{
			name:        "nil request",
			req:         nil,
			setupAuth:   func(ctrl *gomock.Controller) authclient.AuthClient { return NewMockAuthClient(ctrl) },
			expectedErr: ErrUpdateTaskRequestRequired,
		},
		{
			name: "unauthorized session",
			req: &UpdateTaskRequest{
				SessionToken: "invalid-token",
				TaskID:       normalTask.ID().String(),
				UpdateMask:   []string{"title"},
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "invalid-token").
					Return("", authclient.ErrUnauthorized)

				return mockAuth
			},
			expectedErr: ErrUnauthorized,
		},
		{
			name: "empty task id",
			req: &UpdateTaskRequest{
				SessionToken: "token",
				TaskID:       "",
				UpdateMask:   []string{"title"},
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: ErrTaskIDRequired,
		},
		{
			name: "invalid task id format",
			req: &UpdateTaskRequest{
				SessionToken: "token",
				TaskID:       "invalid-uuid",
				UpdateMask:   []string{"title"},
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrIDInvalidFormat,
		},
		{
			name: "task not found",
			req: &UpdateTaskRequest{
				SessionToken: "token",
				TaskID:       missingID.String(),
				UpdateMask:   []string{"title"},
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: ErrTaskNotFound,
		},
		{
			name: "empty update mask",
			req: &UpdateTaskRequest{
				SessionToken: "token",
				TaskID:       normalTask.ID().String(),
				UpdateMask:   []string{},
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrNoFieldsToUpdate,
		},
		{
			name: "invalid field in update mask",
			req: &UpdateTaskRequest{
				SessionToken: "token",
				TaskID:       normalTask.ID().String(),
				UpdateMask:   []string{"task_type"},
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrInvalidUpdateField,
		},
		{
			name: "title too long",
			req: func() *UpdateTaskRequest {
				longTitle := make([]byte, 501)
				for i := range longTitle {
					longTitle[i] = 'a'
				}
				title := string(longTitle)
				return &UpdateTaskRequest{
					SessionToken: "token",
					TaskID:       normalTask.ID().String(),
					UpdateMask:   []string{"title"},
					Title:        &title,
				}
			}(),
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrTitleTooLong,
		},
		{
			name: "scheduled_at on non-SCHEDULED task",
			req: func() *UpdateTaskRequest {
				scheduledAt := time.Now().Add(1 * time.Hour)
				return &UpdateTaskRequest{
					SessionToken: "token",
					TaskID:       normalTask.ID().String(),
					UpdateMask:   []string{"scheduled_at"},
					ScheduledAt:  &scheduledAt,
				}
			}(),
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrScheduledAtNotAllowed,
		},
		{
			name: "invalid color format",
			req: func() *UpdateTaskRequest {
				color := "#FFF"
				return &UpdateTaskRequest{
					SessionToken: "token",
					TaskID:       normalTask.ID().String(),
					UpdateMask:   []string{"color"},
					Color:        &color,
				}
			}(),
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(validUserID.String(), nil)

				return mockAuth
			},
			expectedErr: domaintask.ErrColorInvalidFormat,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAuth := tt.setupAuth(ctrl)
			handler := NewUpdateTaskHandler(mockAuth, repo)

			_, err := handler.UpdateTask(ctx, tt.req)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}

			if !errors.Is(err, tt.expectedErr) && !containsError(err, tt.expectedErr) {
				t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func containsError(err, target error) bool {
	return err != nil && target != nil && (errors.Is(err, target) || (err.Error() != "" && target.Error() != "" && (err.Error() == target.Error() || len(err.Error()) > len(target.Error()) && err.Error()[:len(target.Error())] == target.Error())))
}
