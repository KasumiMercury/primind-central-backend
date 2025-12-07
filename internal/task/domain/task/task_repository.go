package task

import (
	"context"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
)

type TaskRepository interface {
	SaveTask(ctx context.Context, task *Task) error
	GetTaskByID(ctx context.Context, id ID, userID user.ID) (*Task, error)
	ExistsTaskByID(ctx context.Context, id ID) (bool, error)
}
