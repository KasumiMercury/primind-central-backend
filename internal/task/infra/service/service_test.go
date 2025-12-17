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
				Color:    "#FF6B6B",
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

						if req.Color != "#FF6B6B" {
							t.Fatalf("expected color #FF6B6B, got %s", req.Color)
						}

						return &apptask.CreateTaskResult{
							TaskID:   "task-id-1",
							TargetAt: time.Now().Add(1 * time.Hour).UTC(),
							Color:    "#FF6B6B",
						}, nil
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
					Color:       "#4ECDC4",
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

						if req.Color != "#4ECDC4" {
							t.Fatalf("expected color #4ECDC4, got %s", req.Color)
						}

						scheduledTime := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

						return &apptask.CreateTaskResult{
							TaskID:   "task-id-2",
							TargetAt: scheduledTime,
							Color:    "#4ECDC4",
						}, nil
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
					Color:    "#FF6B6B",
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

						if req.Color != "#FF6B6B" {
							t.Fatalf("expected color #FF6B6B, got %s", req.Color)
						}

						return &apptask.CreateTaskResult{
							TaskID:   req.TaskID,
							TargetAt: time.Now().Add(1 * time.Hour).UTC(),
							Color:    "#FF6B6B",
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
			svc := NewService(mockUseCase, nil, nil, nil, nil)

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

			if resp.GetTask().GetTargetAt() == nil {
				t.Fatalf("expected target_at to be set")
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
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil, nil) },
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: "#FF6B6B"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name:    "invalid task type",
			ctx:     ctxWithSessionToken(t, "token"),
			service: func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil, nil) },
			req: &taskv1.CreateTaskRequest{
				Title:    "title",
				TaskType: taskv1.TaskType_TASK_TYPE_UNSPECIFIED,
				Color:    "#FF6B6B",
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

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: "#FF6B6B"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "auth service unavailable",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrAuthServiceUnavailable)

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: "#FF6B6B"},
			expectedCode: connect.CodeUnavailable,
		},
		{
			name: "device service unavailable",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrDeviceServiceUnavailable)

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: "#FF6B6B"},
			expectedCode: connect.CodeUnavailable,
		},
		{
			name: "device invalid argument",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrDeviceInvalidArgument)

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: "#FF6B6B"},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "invalid argument from usecase",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrTitleRequired)

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: "#FF6B6B"},
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

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: "#FF6B6B"},
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

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req: func() *taskv1.CreateTaskRequest {
				invalidUUID := "invalid-uuid"

				return &taskv1.CreateTaskRequest{TaskId: &invalidUUID, Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: "#FF6B6B"}
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

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req: func() *taskv1.CreateTaskRequest {
				uuidv4 := uuid.New()
				uuidv4Str := uuidv4.String()

				return &taskv1.CreateTaskRequest{
					TaskId:   &uuidv4Str,
					Title:    "title",
					TaskType: taskv1.TaskType_TASK_TYPE_NORMAL,
					Color:    "#FF6B6B",
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

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req: func() *taskv1.CreateTaskRequest {
				existingID, _ := domaintask.NewID()
				existingIDStr := existingID.String()

				return &taskv1.CreateTaskRequest{
					TaskId:   &existingIDStr,
					Title:    "title",
					TaskType: taskv1.TaskType_TASK_TYPE_NORMAL,
					Color:    "#FF6B6B",
				}
			}(),
			expectedCode: connect.CodeAlreadyExists,
		},
		{
			name: "empty color returns InvalidArgument",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, domaintask.ErrColorEmpty)

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: ""},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "invalid color format returns InvalidArgument",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockCreateTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					CreateTask(gomock.Any(), gomock.Any()).
					Return(nil, domaintask.ErrColorInvalidFormat)

				return NewService(mockUseCase, nil, nil, nil, nil)
			},
			req:          &taskv1.CreateTaskRequest{Title: "title", TaskType: taskv1.TaskType_TASK_TYPE_NORMAL, Color: "invalid"},
			expectedCode: connect.CodeInvalidArgument,
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
	targetAtNormal := createdAt.Add(1 * time.Hour)
	targetAtScheduled := scheduledAt

	tests := []struct {
		name         string
		req          *taskv1.GetTaskRequest
		expectedCall func(t *testing.T, ctrl *gomock.Controller) apptask.GetTaskUseCase
		targetAt     time.Time
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
							TargetAt:   targetAtNormal,
							Color:      "#FF6B6B",
						}, nil
					})

				return mockUseCase
			},
			targetAt: targetAtNormal,
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
							TargetAt:    targetAtScheduled,
							Color:       "#4ECDC4",
						}, nil
					})

				return mockUseCase
			},
			targetAt: targetAtScheduled,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUseCase := tt.expectedCall(t, ctrl)
			svc := NewService(nil, mockUseCase, nil, nil, nil)
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

			if resp.GetTask().GetTargetAt() == nil {
				t.Fatalf("expected target_at to be set")
			}

			if resp.GetTask().GetTargetAt().AsTime() != tt.targetAt {
				t.Fatalf("expected target at %v, got %v", tt.targetAt, resp.GetTask().GetTargetAt().AsTime())
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
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil, nil) },
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

				return NewService(nil, mockUseCase, nil, nil, nil)
			},
			req:          &taskv1.GetTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "auth service unavailable",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockGetTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					GetTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrAuthServiceUnavailable)

				return NewService(nil, mockUseCase, nil, nil, nil)
			},
			req:          &taskv1.GetTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeUnavailable,
		},
		{
			name: "task not found",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockGetTaskUseCase(ctrl)
				mockUseCase.EXPECT().
					GetTask(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrTaskNotFound)

				return NewService(nil, mockUseCase, nil, nil, nil)
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

				return NewService(nil, mockUseCase, nil, nil, nil)
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

				return NewService(nil, mockUseCase, nil, nil, nil)
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

func TestListActiveTasksSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now().UTC().Truncate(time.Second)

	mockUseCase := NewMockListActiveTasksUseCase(ctrl)
	mockUseCase.EXPECT().
		ListActiveTasks(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *apptask.ListActiveTasksRequest) (*apptask.ListActiveTasksResult, error) {
			if req.SessionToken != "valid-token" {
				t.Fatalf("expected session token valid-token, got %s", req.SessionToken)
			}

			if req.SortType != domaintask.SortTypeTargetAt {
				t.Fatalf("expected sort type target_at, got %s", req.SortType)
			}

			return &apptask.ListActiveTasksResult{
				Tasks: []apptask.TaskItem{
					{
						TaskID:     "task-1",
						Title:      "Task 1",
						TaskType:   domaintask.TypeNormal,
						TaskStatus: domaintask.StatusActive,
						CreatedAt:  now,
						TargetAt:   now.Add(1 * time.Hour),
						Color:      "#FF6B6B",
					},
					{
						TaskID:     "task-2",
						Title:      "Task 2",
						TaskType:   domaintask.TypeUrgent,
						TaskStatus: domaintask.StatusActive,
						CreatedAt:  now,
						TargetAt:   now.Add(30 * time.Minute),
						Color:      "#4ECDC4",
					},
				},
			}, nil
		})

	svc := NewService(nil, nil, mockUseCase, nil, nil)
	ctx := ctxWithSessionToken(t, "valid-token")

	resp, err := svc.ListActiveTasks(ctx, &taskv1.ListActiveTasksRequest{
		SortType: taskv1.TaskSortType_TASK_SORT_TYPE_TARGET_AT,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.GetTasks()) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resp.GetTasks()))
	}

	if resp.GetTasks()[0].GetTaskId() != "task-1" {
		t.Fatalf("expected task id task-1, got %s", resp.GetTasks()[0].GetTaskId())
	}

	if resp.GetTasks()[1].GetTaskId() != "task-2" {
		t.Fatalf("expected task id task-2, got %s", resp.GetTasks()[1].GetTaskId())
	}
}

func TestListActiveTasksError(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		service      func(ctrl *gomock.Controller) *Service
		req          *taskv1.ListActiveTasksRequest
		expectedCode connect.Code
	}{
		{
			name:         "missing session token",
			ctx:          context.Background(),
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil, nil) },
			req:          &taskv1.ListActiveTasksRequest{SortType: taskv1.TaskSortType_TASK_SORT_TYPE_TARGET_AT},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name:         "invalid sort type (unspecified)",
			ctx:          ctxWithSessionToken(t, "token"),
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil, nil) },
			req:          &taskv1.ListActiveTasksRequest{SortType: taskv1.TaskSortType_TASK_SORT_TYPE_UNSPECIFIED},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "unauthorized",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockListActiveTasksUseCase(ctrl)
				mockUseCase.EXPECT().
					ListActiveTasks(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrUnauthorized)

				return NewService(nil, nil, mockUseCase, nil, nil)
			},
			req:          &taskv1.ListActiveTasksRequest{SortType: taskv1.TaskSortType_TASK_SORT_TYPE_TARGET_AT},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "auth service unavailable",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockListActiveTasksUseCase(ctrl)
				mockUseCase.EXPECT().
					ListActiveTasks(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrAuthServiceUnavailable)

				return NewService(nil, nil, mockUseCase, nil, nil)
			},
			req:          &taskv1.ListActiveTasksRequest{SortType: taskv1.TaskSortType_TASK_SORT_TYPE_TARGET_AT},
			expectedCode: connect.CodeUnavailable,
		},
		{
			name: "invalid sort type from use case",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockListActiveTasksUseCase(ctrl)
				mockUseCase.EXPECT().
					ListActiveTasks(gomock.Any(), gomock.Any()).
					Return(nil, apptask.ErrInvalidSortType)

				return NewService(nil, nil, mockUseCase, nil, nil)
			},
			req:          &taskv1.ListActiveTasksRequest{SortType: taskv1.TaskSortType_TASK_SORT_TYPE_TARGET_AT},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "internal error",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockListActiveTasksUseCase(ctrl)
				mockUseCase.EXPECT().
					ListActiveTasks(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))

				return NewService(nil, nil, mockUseCase, nil, nil)
			},
			req:          &taskv1.ListActiveTasksRequest{SortType: taskv1.TaskSortType_TASK_SORT_TYPE_TARGET_AT},
			expectedCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, err := tt.service(ctrl).ListActiveTasks(tt.ctx, tt.req)
			if err == nil {
				t.Fatalf("expected error")
			}

			if connect.CodeOf(err) != tt.expectedCode {
				t.Fatalf("expected code %v, got %v", tt.expectedCode, connect.CodeOf(err))
			}
		})
	}
}

func TestDeleteTaskSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUseCase := NewMockDeleteTaskUseCase(ctrl)
	mockUseCase.EXPECT().
		DeleteTask(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *apptask.DeleteTaskRequest) error {
			if req.SessionToken != "valid-token" {
				t.Fatalf("expected session token valid-token, got %s", req.SessionToken)
			}

			if req.TaskID != "task-id-1" {
				t.Fatalf("expected task id task-id-1, got %s", req.TaskID)
			}

			return nil
		})

	svc := NewService(nil, nil, nil, nil, mockUseCase)
	ctx := ctxWithSessionToken(t, "valid-token")

	resp, err := svc.DeleteTask(ctx, &taskv1.DeleteTaskRequest{TaskId: "task-id-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatalf("expected response")
	}
}

func TestDeleteTaskError(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		service      func(ctrl *gomock.Controller) *Service
		req          *taskv1.DeleteTaskRequest
		expectedCode connect.Code
	}{
		{
			name:         "missing session token",
			ctx:          context.Background(),
			service:      func(_ *gomock.Controller) *Service { return NewService(nil, nil, nil, nil, nil) },
			req:          &taskv1.DeleteTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "unauthorized",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockDeleteTaskUseCase(ctrl)
				mockUseCase.EXPECT().DeleteTask(gomock.Any(), gomock.Any()).Return(apptask.ErrUnauthorized)

				return NewService(nil, nil, nil, nil, mockUseCase)
			},
			req:          &taskv1.DeleteTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeUnauthenticated,
		},
		{
			name: "auth service unavailable",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockDeleteTaskUseCase(ctrl)
				mockUseCase.EXPECT().DeleteTask(gomock.Any(), gomock.Any()).Return(apptask.ErrAuthServiceUnavailable)

				return NewService(nil, nil, nil, nil, mockUseCase)
			},
			req:          &taskv1.DeleteTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeUnavailable,
		},
		{
			name: "task not found",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockDeleteTaskUseCase(ctrl)
				mockUseCase.EXPECT().DeleteTask(gomock.Any(), gomock.Any()).Return(apptask.ErrTaskNotFound)

				return NewService(nil, nil, nil, nil, mockUseCase)
			},
			req:          &taskv1.DeleteTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeNotFound,
		},
		{
			name: "invalid argument - empty task id",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockDeleteTaskUseCase(ctrl)
				mockUseCase.EXPECT().DeleteTask(gomock.Any(), gomock.Any()).Return(apptask.ErrTaskIDRequired)

				return NewService(nil, nil, nil, nil, mockUseCase)
			},
			req:          &taskv1.DeleteTaskRequest{TaskId: ""},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "invalid argument - invalid id format",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockDeleteTaskUseCase(ctrl)
				mockUseCase.EXPECT().DeleteTask(gomock.Any(), gomock.Any()).Return(domaintask.ErrIDInvalidFormat)

				return NewService(nil, nil, nil, nil, mockUseCase)
			},
			req:          &taskv1.DeleteTaskRequest{TaskId: "invalid-uuid"},
			expectedCode: connect.CodeInvalidArgument,
		},
		{
			name: "internal error",
			ctx:  ctxWithSessionToken(t, "token"),
			service: func(ctrl *gomock.Controller) *Service {
				mockUseCase := NewMockDeleteTaskUseCase(ctrl)
				mockUseCase.EXPECT().DeleteTask(gomock.Any(), gomock.Any()).Return(errors.New("boom"))

				return NewService(nil, nil, nil, nil, mockUseCase)
			},
			req:          &taskv1.DeleteTaskRequest{TaskId: "id"},
			expectedCode: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_, err := tt.service(ctrl).DeleteTask(tt.ctx, tt.req)
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
