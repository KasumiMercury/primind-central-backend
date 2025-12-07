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
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCreateTaskSuccess(t *testing.T) {
	tests := []struct {
		name         string
		req          *taskv1.CreateTaskRequest
		expectedCall func(t *testing.T, ctrl *gomock.Controller) apptask.CreateTaskUseCase
	}{
		{
			name: "normal task without scheduled time",
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

						if req.Description != "" {
							t.Fatalf("expected empty description, got %v", req.Description)
						}

						if req.ScheduledAt != nil {
							t.Fatalf("expected nil scheduled time, got %v", req.ScheduledAt)
						}

						return &apptask.CreateTaskResult{TaskID: "task-id-1"}, nil
					})

				return mockUseCase
			},
		},
		{
			name: "task with description and scheduled time",
			req: func() *taskv1.CreateTaskRequest {
				desc := "desc"
				scheduledAt := timestamppb.New(time.Now().Add(time.Hour).UTC().Truncate(time.Second))

				return &taskv1.CreateTaskRequest{
					Title:       "task with scheduled",
					TaskType:    taskv1.TaskType_TASK_TYPE_SCHEDULED,
					Description: desc,
					ScheduledAt: scheduledAt,
				}
			}(),
			expectedCall: func(t *testing.T, ctrl *gomock.Controller) apptask.CreateTaskUseCase {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().CreateTask(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *apptask.CreateTaskRequest) (*apptask.CreateTaskResult, error) {
						if req.SessionToken != "token-scheduled" {
							t.Fatalf("expected session token token-scheduled, got %s", req.SessionToken)
						}

						if req.Title != "task with scheduled" {
							t.Fatalf("expected title task with scheduled, got %s", req.Title)
						}

						if req.TaskType != domaintask.TypeScheduled {
							t.Fatalf("expected task type %s, got %s", domaintask.TypeScheduled, req.TaskType)
						}

						if req.Description != "desc" {
							t.Fatalf("unexpected description: %v", req.Description)
						}

						if req.ScheduledAt == nil {
							t.Fatalf("expected scheduled time to be set")
						}

						if req.ScheduledAt.UTC().Truncate(time.Second) != req.ScheduledAt.UTC() {
							t.Fatalf("scheduled time should be utc second precision, got %v", req.ScheduledAt)
						}

						return &apptask.CreateTaskResult{TaskID: "task-id-2"}, nil
					})

				return mockUseCase
			},
		},
		{
			name: "create task with predefined task ID",
			req: func() *taskv1.CreateTaskRequest {
				validTaskID, _ := domaintask.NewID()
				taskIDStr := validTaskID.String()

				return &taskv1.CreateTaskRequest{
					TaskId:   &taskIDStr,
					Title:    "Task with predefined ID",
					TaskType: taskv1.TaskType_TASK_TYPE_NORMAL,
				}
			}(),
			expectedCall: func(t *testing.T, ctrl *gomock.Controller) apptask.CreateTaskUseCase {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().CreateTask(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *apptask.CreateTaskRequest) (*apptask.CreateTaskResult, error) {
						if req.SessionToken != "token-with-id" {
							t.Fatalf("expected session token token-with-id, got %s", req.SessionToken)
						}

						if req.TaskID == "" {
							t.Fatal("expected TaskID to be set")
						}

						// Verify TaskID is valid UUIDv7
						_, err := domaintask.NewIDFromString(req.TaskID)
						if err != nil {
							t.Fatalf("expected valid TaskID, got error: %v", err)
						}

						if req.Title != "Task with predefined ID" {
							t.Fatalf("expected title Task with predefined ID, got %s", req.Title)
						}

						return &apptask.CreateTaskResult{TaskID: req.TaskID}, nil
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
			if tt.name == "task with description and scheduled time" {
				token = "token-scheduled"
			}

			if tt.name == "create task with predefined task ID" {
				token = "token-with-id"
			}

			ctx := ctxWithSessionToken(t, token)

			resp, err := svc.CreateTask(ctx, tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.GetTask().GetTaskId() == "" {
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
		{
			name: "invalid task ID format returns InvalidArgument",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, domaintask.ErrIDInvalidFormat)

				return NewService(mockUseCase, nil)
			},
			req: func() *taskv1.CreateTaskRequest {
				invalidUUID := "invalid-uuid"

				return &taskv1.CreateTaskRequest{TaskId: &invalidUUID, Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL}
			}(),
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "task ID not v7 returns InvalidArgument",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, domaintask.ErrIDInvalidV7)

				return NewService(mockUseCase, nil)
			},
			req: func() *taskv1.CreateTaskRequest {
				uuidv4 := uuid.New()
				uuidv4Str := uuidv4.String()

				return &taskv1.CreateTaskRequest{
					TaskId:   &uuidv4Str,
					Title:    "title",
					TaskType: taskv1.TaskType_TASK_TYPE_NORMAL,
				}
			}(),
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "duplicate task ID returns AlreadyExists",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrTaskIDAlreadyExists)

				return NewService(mockUseCase, nil)
			},
			req: func() *taskv1.CreateTaskRequest {
				existingID, _ := domaintask.NewID()
				existingIDStr := existingID.String()

				return &taskv1.CreateTaskRequest{
					TaskId:   &existingIDStr,
					Title:    "title",
					TaskType: taskv1.TaskType_TASK_TYPE_NORMAL,
				}
			}(),
			expectedCode: connect.CodeAlreadyExists,
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
	scheduledAt := createdAt.Add(30 * time.Minute)

	tests := []struct {
		name         string
		req          *taskv1.GetTaskRequest
		expectedCall func(t *testing.T, ctrl *gomock.Controller) apptask.GetTaskUseCase
	}{
		{
			name: "get task without scheduled time",
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
			name: "get task with scheduled time and description",
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
							TaskType:    domaintask.TypeScheduled,
							TaskStatus:  domaintask.StatusCompleted,
							Description: desc,
							ScheduledAt: &scheduledAt,
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

			if resp.GetTask().GetTaskId() == "" {
				t.Fatalf("expected task id")
			}

			if resp.GetTask().GetTitle() == "" {
				t.Fatalf("expected title")
			}

			if resp.GetTask().GetCreatedAt().AsTime() != createdAt {
				t.Fatalf("expected created at %v, got %v", createdAt, resp.GetTask().GetCreatedAt().AsTime())
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
