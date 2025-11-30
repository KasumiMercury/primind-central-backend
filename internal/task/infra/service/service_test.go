package task

import (
	"context"
	"errors"
	"testing"
	"time"

	connect "connectrpc.com/connect"
	taskv1 "github.com/KasumiMercury/primind-central-backend/internal/gen/task/v1"
	apptask "github.com/KasumiMercury/primind-central-backend/internal/task/app/task"
	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/interceptor"
	"go.uber.org/mock/gomock"
)

func TestCreateTaskSuccess(t *testing.T) {
	tests := []struct {
		name         string
		req          *taskv1.CreateTaskRequest
		expectedCall func(t *testing.T, ctrl *gomock.Controller) apptask.CreateTaskUseCase
	}{
		{
			name: "normal task without due time",
			req: &taskv1.CreateTaskRequest{
				Title:    "task title",
				TaskType: taskv1.TaskType_TASK_TYPE_NORMAL,
			},
			expectedCall: func(t *testing.T, ctrl *gomock.Controller) apptask.CreateTaskUseCase {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().CreateTask(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *apptask.CreateTaskRequest) (*apptask.CreateTaskResult, error) {
						if req.SessionToken != "token-normal" {
							t.Fatalf("expected session token token-normal, got %s", req.SessionToken)
						}

						if req.Title != "task title" {
							t.Fatalf("expected title task title, got %s", req.Title)
						}

						if req.TaskType != domaintask.TypeNormal {
							t.Fatalf("expected task type %s, got %s", domaintask.TypeNormal, req.TaskType)
						}

						if req.Description != nil {
							t.Fatalf("expected nil description, got %v", req.Description)
						}

						if req.DueTime != nil {
							t.Fatalf("expected nil due time, got %v", req.DueTime)
						}

						return &apptask.CreateTaskResult{TaskID: "task-id-1"}, nil
					})

				return mockUseCase
			},
		},
		{
			name: "task with description and due time",
			req: func() *taskv1.CreateTaskRequest {
				desc := "desc"
				dueTime := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
				dueUnix := dueTime.Unix()

				return &taskv1.CreateTaskRequest{
					Title:       "task with due",
					TaskType:    taskv1.TaskType_TASK_TYPE_HAS_DUE_TIME,
					Description: &desc,
					DueTime:     &dueUnix,
				}
			}(),
			expectedCall: func(t *testing.T, ctrl *gomock.Controller) apptask.CreateTaskUseCase {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().CreateTask(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *apptask.CreateTaskRequest) (*apptask.CreateTaskResult, error) {
						if req.SessionToken != "token-due" {
							t.Fatalf("expected session token token-due, got %s", req.SessionToken)
						}

						if req.Title != "task with due" {
							t.Fatalf("expected title task with due, got %s", req.Title)
						}

						if req.TaskType != domaintask.TypeHasDueTime {
							t.Fatalf("expected task type %s, got %s", domaintask.TypeHasDueTime, req.TaskType)
						}

						if req.Description == nil || *req.Description != "desc" {
							t.Fatalf("unexpected description: %v", req.Description)
						}

						if req.DueTime == nil {
							t.Fatalf("expected due time to be set")
						}

						if req.DueTime.UTC().Truncate(time.Second) != req.DueTime.UTC() {
							t.Fatalf("due time should be utc second precision, got %v", req.DueTime)
						}

						return &apptask.CreateTaskResult{TaskID: "task-id-2"}, nil
					})

				return mockUseCase
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUseCase := tt.expectedCall(t, ctrl)
			svc := NewService(mockUseCase, nil)

			token := "token-normal"
			if tt.name == "task with description and due time" {
				token = "token-due"
			}

			ctx := ctxWithSessionToken(t, token)

			resp, err := svc.CreateTask(ctx, tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.GetTaskId() == "" {
				t.Fatalf("expected task id to be set")
			}
		})
	}
}

func TestCreateTaskError(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		service      func(ctrl *gomock.Controller) *Service
		req          *taskv1.CreateTaskRequest
		expectedCode connect.Code
	}{
		{
			name:         "missing session token",
			ctx:          context.Background(),
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil) },
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name:    "invalid task type",
			ctx:     ctxWithSessionToken(t, "token"),
			service: func(_ *gomock.Controller) *Service { return NewService(nil, nil) },
			req: &taskv1.CreateTaskRequest{
				Title:    "title",
				TaskType: taskv1.TaskType_TASK_TYPE_UNSPECIFIED,
			},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "unauthorized create task",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrUnauthorized)

				return NewService(mockUseCase, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "invalid argument from usecase",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrTitleRequired)

				return NewService(mockUseCase, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "internal error",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("boom"))

				return NewService(mockUseCase, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL},
			expectedCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, err := tt.service(ctrl).CreateTask(tt.ctx, tt.req)
			if err == nil {
				t.Fatalf("expected error")
			}

			if connect.CodeOf(err) != tt.expectedCode {
				t.Fatalf("expected code %v, got %v", tt.expectedCode, connect.CodeOf(err))
			}
		})
	}
}

func TestGetTaskSuccess(t *testing.T) {
	createdAt := time.Now().UTC().Truncate(time.Second)
	dueTime := createdAt.Add(30 * time.Minute)

	tests := []struct {
		name         string
		req          *taskv1.GetTaskRequest
		expectedCall func(t *testing.T, ctrl *gomock.Controller) apptask.GetTaskUseCase
	}{
		{
			name: "get task without due time",
			req:  &taskv1.GetTaskRequest{TaskId: "task-id-1"},
			expectedCall: func(t *testing.T, ctrl *gomock.Controller) apptask.GetTaskUseCase {
				mockUseCase := NewMockGetTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					GetTask(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *apptask.GetTaskRequest) (*apptask.GetTaskResult, error) {
						if req.SessionToken != "token" {
							t.Fatalf("expected session token token, got %s", req.SessionToken)
						}

						if req.TaskID != "task-id-1" {
							t.Fatalf("expected task id task-id-1, got %s", req.TaskID)
						}

						return &apptask.GetTaskResult{
							TaskID:     "task-id-1",
							Title:      "title",
							TaskType:   domaintask.TypeNormal,
							TaskStatus: domaintask.StatusActive,
							CreatedAt:  createdAt,
						}, nil
					})

				return mockUseCase
			},
		},
		{
			name: "get task with due time and description",
			req:  &taskv1.GetTaskRequest{TaskId: "task-id-2"},
			expectedCall: func(t *testing.T, ctrl *gomock.Controller) apptask.GetTaskUseCase {
				mockUseCase := NewMockGetTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					GetTask(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *apptask.GetTaskRequest) (*apptask.GetTaskResult, error) {
						if req.SessionToken != "token" {
							t.Fatalf("expected session token token, got %s", req.SessionToken)
						}

						if req.TaskID != "task-id-2" {
							t.Fatalf("expected task id task-id-2, got %s", req.TaskID)
						}

						desc := "desc"

						return &apptask.GetTaskResult{
							TaskID:      "task-id-2",
							Title:       "title 2",
							TaskType:    domaintask.TypeHasDueTime,
							TaskStatus:  domaintask.StatusCompleted,
							Description: &desc,
							DueTime:     &dueTime,
							CreatedAt:   createdAt,
						}, nil
					})

				return mockUseCase
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUseCase := tt.expectedCall(t, ctrl)
			svc := NewService(nil, mockUseCase)
			ctx := ctxWithSessionToken(t, "token")

			resp, err := svc.GetTask(ctx, tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.GetTaskId() == "" {
				t.Fatalf("expected task id")
			}

			if resp.GetTitle() == "" {
				t.Fatalf("expected title")
			}

			if resp.GetCreatedAt() != createdAt.Unix() {
				t.Fatalf("expected created at %v, got %v", createdAt.Unix(), resp.GetCreatedAt())
			}
		})
	}
}

func TestGetTaskError(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		service      func(ctrl *gomock.Controller) *Service
		req          *taskv1.GetTaskRequest
		expectedCode connect.Code
	}{
		{
			name:         "missing session token",
			ctx:          context.Background(),
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil) },
			req:          &taskv1.GetTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "unauthorized get task",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockGetTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					GetTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrUnauthorized)

				return NewService(nil, mockUseCase)
			},
			req:          &taskv1.GetTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "task not found",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockGetTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					GetTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrTaskNotFound)

				return NewService(nil, mockUseCase)
			},
			req:          &taskv1.GetTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeNotFound,
		},
		{
			name: "invalid argument",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockGetTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					GetTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrTaskIDRequired)

				return NewService(nil, mockUseCase)
			},
			req:          &taskv1.GetTaskRequest{TaskId: ""},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "internal error",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockGetTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					GetTask(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("boom"))

				return NewService(nil, mockUseCase)
			},
			req:          &taskv1.GetTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, err := tt.service(ctrl).GetTask(tt.ctx, tt.req)
			if err == nil {
				t.Fatalf("expected error")
			}

			if connect.CodeOf(err) != tt.expectedCode {
				t.Fatalf("expected code %v, got %v", tt.expectedCode, connect.CodeOf(err))
			}
		})
	}
}

func ctxWithSessionToken(t *testing.T, token string) context.Context {
	t.Helper()

	req := connect.NewRequest(&taskv1.CreateTaskRequest{})
	req.Header().Set("Authorization", "Bearer "+token)

	var capturedCtx context.Context

	_, err := interceptor.AuthInterceptor()(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		capturedCtx = ctx

		return connect.NewResponse(&taskv1.CreateTaskResponse{}), nil
	})(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to prepare context with token: %v", err)
	}

	return capturedCtx
}
