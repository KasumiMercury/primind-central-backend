package task

import (
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
)

type CompletedTask struct {
	id          ID
	userID      user.ID
	title       string
	taskType    Type
	description string
	scheduledAt *time.Time
	createdAt   time.Time
	targetAt    time.Time
	color       Color
	completedAt time.Time
}

func NewCompletedTask(task *Task, completedAt time.Time) (*CompletedTask, error) {
	if task == nil {
		return nil, ErrTaskNil
	}

	return &CompletedTask{
		id:          task.ID(),
		userID:      task.UserID(),
		title:       task.Title(),
		taskType:    task.TaskType(),
		description: task.Description(),
		scheduledAt: task.ScheduledAt(),
		createdAt:   task.CreatedAt(),
		targetAt:    task.TargetAt(),
		color:       task.Color(),
		completedAt: completedAt.UTC().Truncate(time.Microsecond),
	}, nil
}

func (ct *CompletedTask) ID() ID {
	return ct.id
}

func (ct *CompletedTask) UserID() user.ID {
	return ct.userID
}

func (ct *CompletedTask) Title() string {
	return ct.title
}

func (ct *CompletedTask) TaskType() Type {
	return ct.taskType
}

func (ct *CompletedTask) Description() string {
	return ct.description
}

func (ct *CompletedTask) ScheduledAt() *time.Time {
	return ct.scheduledAt
}

func (ct *CompletedTask) CreatedAt() time.Time {
	return ct.createdAt
}

func (ct *CompletedTask) TargetAt() time.Time {
	return ct.targetAt
}

func (ct *CompletedTask) Color() Color {
	return ct.color
}

func (ct *CompletedTask) CompletedAt() time.Time {
	return ct.completedAt
}
