package task

import "context"

type TaskRepository interface {
	SaveTask(ctx context.Context, task *Task) error
	GetTaskByID(ctx context.Context, id ID, userID string) (*Task, error)
}
