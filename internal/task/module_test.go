package task

import (
	"context"
	"testing"

	apptask "github.com/KasumiMercury/primind-central-backend/internal/task/app/task"
	domaintask "github.com/KasumiMercury/primind-central-backend/internal/task/domain/task"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindcancel"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/remindregister"
	"github.com/KasumiMercury/primind-central-backend/internal/task/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/testutil"
	"go.uber.org/mock/gomock"
)

func setupTaskRepo(t *testing.T) domaintask.TaskRepository {
	t.Helper()

	ctx := context.Background()
	db, cleanup := testutil.SetupPostgresContainer(ctx, t)
	t.Cleanup(cleanup)

	if err := db.AutoMigrate(&repository.TaskModel{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return repository.NewTaskRepository(db)
}

func TestNewHTTPHandlerWithRepositoriesSuccess(t *testing.T) {
	repo := setupTaskRepo(t)
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	path, handler, err := NewHTTPHandlerWithRepositories(context.Background(), Repositories{
		Tasks:               repo,
		TaskArchive:         domaintask.NewMockTaskArchiveRepository(ctrl),
		AuthClient:          apptask.NewMockAuthClient(ctrl),
		DeviceClient:        apptask.NewMockDeviceClient(ctrl),
		RemindRegisterQueue: remindregister.NewMockQueue(ctrl),
		RemindCancelQueue:   remindcancel.NewMockQueue(ctrl),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if path == "" {
		t.Fatalf("expected path to be set")
	}

	if handler == nil {
		t.Fatalf("expected handler to be non-nil")
	}
}

func TestNewHTTPHandlerWithRepositoriesError(t *testing.T) {
	tests := []struct {
		name        string
		repos       func(t *testing.T) Repositories
		ctx         context.Context
		expectError bool
	}{
		{
			name: "missing repository",
			repos: func(t *testing.T) Repositories {
				ctrl := gomock.NewController(t)
				t.Cleanup(ctrl.Finish)

				return Repositories{
					Tasks:               nil,
					TaskArchive:         domaintask.NewMockTaskArchiveRepository(ctrl),
					AuthClient:          apptask.NewMockAuthClient(ctrl),
					DeviceClient:        apptask.NewMockDeviceClient(ctrl),
					RemindRegisterQueue: remindregister.NewMockQueue(ctrl),
					RemindCancelQueue:   remindcancel.NewMockQueue(ctrl),
				}
			},
			ctx:         context.Background(),
			expectError: true,
		},
		{
			name: "missing task archive repository",
			repos: func(t *testing.T) Repositories {
				ctrl := gomock.NewController(t)
				t.Cleanup(ctrl.Finish)

				return Repositories{
					Tasks:               setupTaskRepo(t),
					TaskArchive:         nil,
					AuthClient:          apptask.NewMockAuthClient(ctrl),
					DeviceClient:        apptask.NewMockDeviceClient(ctrl),
					RemindRegisterQueue: remindregister.NewMockQueue(ctrl),
					RemindCancelQueue:   remindcancel.NewMockQueue(ctrl),
				}
			},
			ctx:         context.Background(),
			expectError: true,
		},
		{
			name: "missing auth client",
			repos: func(t *testing.T) Repositories {
				ctrl := gomock.NewController(t)
				t.Cleanup(ctrl.Finish)

				return Repositories{
					Tasks:               setupTaskRepo(t),
					TaskArchive:         domaintask.NewMockTaskArchiveRepository(ctrl),
					AuthClient:          nil,
					DeviceClient:        apptask.NewMockDeviceClient(ctrl),
					RemindRegisterQueue: remindregister.NewMockQueue(ctrl),
					RemindCancelQueue:   remindcancel.NewMockQueue(ctrl),
				}
			},
			ctx:         context.Background(),
			expectError: true,
		},
		{
			name: "missing device client",
			repos: func(t *testing.T) Repositories {
				ctrl := gomock.NewController(t)
				t.Cleanup(ctrl.Finish)

				return Repositories{
					Tasks:               setupTaskRepo(t),
					TaskArchive:         domaintask.NewMockTaskArchiveRepository(ctrl),
					AuthClient:          apptask.NewMockAuthClient(ctrl),
					DeviceClient:        nil,
					RemindRegisterQueue: remindregister.NewMockQueue(ctrl),
					RemindCancelQueue:   remindcancel.NewMockQueue(ctrl),
				}
			},
			ctx:         context.Background(),
			expectError: true,
		},
		{
			name: "missing remind register queue",
			repos: func(t *testing.T) Repositories {
				ctrl := gomock.NewController(t)
				t.Cleanup(ctrl.Finish)

				return Repositories{
					Tasks:               setupTaskRepo(t),
					TaskArchive:         domaintask.NewMockTaskArchiveRepository(ctrl),
					AuthClient:          apptask.NewMockAuthClient(ctrl),
					DeviceClient:        apptask.NewMockDeviceClient(ctrl),
					RemindRegisterQueue: nil,
					RemindCancelQueue:   remindcancel.NewMockQueue(ctrl),
				}
			},
			ctx:         context.Background(),
			expectError: true,
		},
		{
			name: "missing remind cancel queue",
			repos: func(t *testing.T) Repositories {
				ctrl := gomock.NewController(t)
				t.Cleanup(ctrl.Finish)

				return Repositories{
					Tasks:               setupTaskRepo(t),
					TaskArchive:         domaintask.NewMockTaskArchiveRepository(ctrl),
					AuthClient:          apptask.NewMockAuthClient(ctrl),
					DeviceClient:        apptask.NewMockDeviceClient(ctrl),
					RemindRegisterQueue: remindregister.NewMockQueue(ctrl),
					RemindCancelQueue:   nil,
				}
			},
			ctx:         context.Background(),
			expectError: true,
		},
		{
			name: "context canceled",
			repos: func(t *testing.T) Repositories {
				ctrl := gomock.NewController(t)
				t.Cleanup(ctrl.Finish)

				return Repositories{
					Tasks:               setupTaskRepo(t),
					TaskArchive:         domaintask.NewMockTaskArchiveRepository(ctrl),
					AuthClient:          apptask.NewMockAuthClient(ctrl),
					DeviceClient:        apptask.NewMockDeviceClient(ctrl),
					RemindRegisterQueue: remindregister.NewMockQueue(ctrl),
					RemindCancelQueue:   remindcancel.NewMockQueue(ctrl),
				}
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := NewHTTPHandlerWithRepositories(tt.ctx, tt.repos(t))
			if tt.expectError && err == nil {
				t.Fatalf("expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
