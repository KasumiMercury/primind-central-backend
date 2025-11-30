package task

import (
	"context"
	"errors"
	"testing"
	"time"

	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/authclient"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/testutil"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestCreateTaskSuccess(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		req    CreateTaskRequest
		userID string
	}{
		{
			name: "create normal task without due time",
			req: CreateTaskRequest{
				SessionToken: "token-normal",
				Title:        "Test Task",
				TaskType:     domaintask.TypeNormal,
			},
			userID: uuid.NewString(),
		},
		{
			name: "create task with due time and description",
			req: func() CreateTaskRequest {
				desc := "task description"
				due := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)

				return CreateTaskRequest{
					SessionToken: "token-due",
					Title:        "Task with due time",
					TaskType:     domaintask.TypeHasDueTime,
					Description:  &desc,
					DueTime:      &due,
				}
			}(),
			userID: uuid.NewString(),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAuth := NewMockAuthClient(ctrl)
			mockAuth.EXPECT().ValidateSession(gomock.Any(), tt.req.SessionToken).Return(tt.userID, nil)

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

			if tt.req.Description == nil && saved.Description() != nil {
				t.Fatalf("expected nil description, got %v", saved.Description())
			}

			if tt.req.Description != nil && (saved.Description() == nil || *saved.Description() != *tt.req.Description) {
				t.Fatalf("expected description %q, got %v", *tt.req.Description, saved.Description())
			}

			if tt.req.DueTime == nil && saved.DueTime() != nil {
				t.Fatalf("expected nil due time, got %v", saved.DueTime())
			}

			if tt.req.DueTime != nil {
				if saved.DueTime() == nil {
					t.Fatalf("expected due time, got nil")
				}

				if !saved.DueTime().Equal(*tt.req.DueTime) {
					t.Fatalf("expected due time %v, got %v", tt.req.DueTime, saved.DueTime())
				}
			}
		})
	}
}

func TestCreateTaskError(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

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
			name: "empty title",
			req: &CreateTaskRequest{
				SessionToken: "token",
				Title:        "",
				TaskType:     domaintask.TypeNormal,
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(uuid.NewString(), nil)
				return mockAuth
			},
			expectedErr: ErrTitleRequired,
		},
		{
			name: "due time required for has_due_time",
			req: &CreateTaskRequest{
				SessionToken: "token",
				Title:        "task without due time",
				TaskType:     domaintask.TypeHasDueTime,
			},
			setupAuth: func(ctrl *gomock.Controller) authclient.AuthClient {
				mockAuth := NewMockAuthClient(ctrl)
				mockAuth.EXPECT().ValidateSession(gomock.Any(), "token").
					Return(uuid.NewString(), nil)
				return mockAuth
			},
			expectedErr: domaintask.ErrDueTimeRequired,
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
	due := time.Now().Add(3 * time.Hour).UTC().Truncate(time.Second)

	userIDNormal := uuid.NewString()
	userIDWithDue := uuid.NewString()

	now := time.Now().UTC()

	taskWithNoDue := createPersistedTask(t, repo, userIDNormal, "stored", domaintask.TypeNormal, nil, nil, now)
	taskWithDue := createPersistedTask(t, repo, userIDWithDue, "stored with due", domaintask.TypeHasDueTime, &desc, &due, now)

	tests := []struct {
		name         string
		req          GetTaskRequest
		userID       string
		expectedTask *domaintask.Task
	}{
		{
			name: "get normal task",
			req: GetTaskRequest{
				SessionToken: "token-normal",
				TaskID:       taskWithNoDue.ID().String(),
			},
			userID:       userIDNormal,
			expectedTask: taskWithNoDue,
		},
		{
			name: "get task with due time",
			req: GetTaskRequest{
				SessionToken: "token-due",
				TaskID:       taskWithDue.ID().String(),
			},
			userID:       userIDWithDue,
			expectedTask: taskWithDue,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockAuth := NewMockAuthClient(ctrl)
			mockAuth.EXPECT().ValidateSession(gomock.Any(), tt.req.SessionToken).
				Return(tt.userID, nil)

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

			if tt.expectedTask.Description() == nil && resp.Description != nil {
				t.Fatalf("expected nil description, got %v", resp.Description)
			}

			if tt.expectedTask.Description() != nil {
				if resp.Description == nil || *resp.Description != *tt.expectedTask.Description() {
					t.Fatalf("expected description %q, got %v", *tt.expectedTask.Description(), resp.Description)
				}
			}

			if tt.expectedTask.DueTime() == nil && resp.DueTime != nil {
				t.Fatalf("expected nil due time, got %v", resp.DueTime)
			}

			if tt.expectedTask.DueTime() != nil {
				if resp.DueTime == nil || !resp.DueTime.Equal(*tt.expectedTask.DueTime()) {
					t.Fatalf("expected due time %v, got %v", tt.expectedTask.DueTime(), resp.DueTime)
				}
			}

			if !resp.CreatedAt.Equal(tt.expectedTask.CreatedAt()) {
				t.Fatalf("expected created at %v, got %v", tt.expectedTask.CreatedAt(), resp.CreatedAt)
			}
		})
	}
}

func TestGetTaskError(t *testing.T) {
	repo := setupTaskRepository(t)
	ctx := context.Background()

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
					Return(uuid.NewString(), nil)
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
					Return(uuid.NewString(), nil)
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
					Return(uuid.NewString(), nil)
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
	userID string,
	title string,
	taskType domaintask.Type,
	description *string,
	dueTime *time.Time,
	createdAt time.Time,
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
		domaintask.StatusActive,
		description,
		dueTime,
		createdAt,
	)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := repo.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	return task
}
