package task

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/KasumiMercury/primind-central-backend/internal/task/domain/user"
	"github.com/google/uuid"
)

type ID uuid.UUID

func NewID() (ID, error) {
	v7, err := uuid.NewV7()
	if err != nil {
		return ID{}, fmt.Errorf("%w: %v", ErrIDGeneration, err)
	}

	return ID(v7), nil
}

func NewIDFromString(idStr string) (ID, error) {
	uuidVal, err := uuid.Parse(idStr)
	if err != nil {
		return ID{}, fmt.Errorf("%w: %v", ErrIDInvalidFormat, err)
	}

	if uuidVal.Version() != 7 {
		return ID{}, ErrIDInvalidV7
	}

	return ID(uuidVal), nil
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

type Type string

const (
	TypeUrgent    Type = "urgent"
	TypeNormal    Type = "normal"
	TypeLow       Type = "low"
	TypeScheduled Type = "scheduled"
)

func NewType(t string) (Type, error) {
	switch t {
	case string(TypeUrgent), string(TypeNormal), string(TypeLow), string(TypeScheduled):
		return Type(t), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidTaskType, t)
	}
}

type Status string

const (
	StatusActive           Status = "active"
	StatusCompleted        Status = "completed"
	StatusPendingReminders Status = "pending_reminders"
)

func NewStatus(s string) (Status, error) {
	switch s {
	case string(StatusActive), string(StatusCompleted), string(StatusPendingReminders):
		return Status(s), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidTaskStatus, s)
	}
}

type Task struct {
	id          ID
	userID      user.ID
	title       string
	taskType    Type
	taskStatus  Status
	description string
	scheduledAt *time.Time
	createdAt   time.Time
	targetAt    time.Time
	color       Color
}

func NewTask(
	id ID,
	userID user.ID,
	title string,
	taskType Type,
	taskStatus Status,
	description string,
	scheduledAt *time.Time,
	createdAt time.Time,
	targetAt time.Time,
	color Color,
) (*Task, error) {
	normalizedCreatedAt := createdAt.UTC().Truncate(time.Microsecond)
	normalizedTargetAt := targetAt.UTC().Truncate(time.Microsecond)

	normalizedScheduledAt := scheduledAt
	if scheduledAt != nil {
		t := scheduledAt.UTC().Truncate(time.Microsecond)
		normalizedScheduledAt = &t
	}

	if utf8.RuneCountInString(title) > 500 {
		return nil, ErrTitleTooLong
	}

	if taskType == TypeScheduled {
		if normalizedScheduledAt == nil {
			return nil, ErrScheduledAtRequired
		}

		if normalizedScheduledAt.Before(normalizedCreatedAt) {
			return nil, ErrScheduledAtBeforeCreatedAt
		}
	}

	if taskType != TypeScheduled && normalizedScheduledAt != nil {
		return nil, ErrScheduledAtNotAllowed
	}

	if err := color.Validate(); err != nil {
		return nil, err
	}

	return &Task{
		id:          id,
		userID:      userID,
		title:       title,
		taskType:    taskType,
		taskStatus:  taskStatus,
		description: description,
		scheduledAt: normalizedScheduledAt,
		createdAt:   normalizedCreatedAt,
		targetAt:    normalizedTargetAt,
		color:       color,
	}, nil
}

func CreateTask(
	taskID *ID,
	userID user.ID,
	title string,
	taskType Type,
	description string,
	scheduledAt *time.Time,
	color Color,
) (*Task, error) {
	var id ID

	if taskID != nil {
		id = *taskID
	} else {
		newID, err := NewID()
		if err != nil {
			return nil, err
		}

		id = newID
	}

	createdAt := time.Now().UTC()

	var targetAt time.Time

	if taskType == TypeScheduled {
		if scheduledAt != nil {
			targetAt = *scheduledAt
		}
	} else {
		activePeriod := GetActivePeriodForType(taskType)
		targetAt = createdAt.Add(time.Duration(activePeriod))
	}

	return NewTask(
		id,
		userID,
		title,
		taskType,
		StatusPendingReminders,
		description,
		scheduledAt,
		createdAt,
		targetAt,
		color,
	)
}

func (t *Task) ID() ID {
	return t.id
}

func (t *Task) UserID() user.ID {
	return t.userID
}

func (t *Task) Title() string {
	return t.title
}

func (t *Task) TaskType() Type {
	return t.taskType
}

func (t *Task) TaskStatus() Status {
	return t.taskStatus
}

func (t *Task) Description() string {
	return t.description
}

func (t *Task) ScheduledAt() *time.Time {
	return t.scheduledAt
}

func (t *Task) CreatedAt() time.Time {
	return t.createdAt
}

func (t *Task) TargetAt() time.Time {
	return t.targetAt
}

func (t *Task) Color() Color {
	return t.color
}

type TaskUpdateInput struct {
	TaskStatus       *Status
	Title            *string
	Description      *string
	ScheduledAt      *time.Time
	ClearScheduledAt bool
	Color            *Color
}

func (u *TaskUpdateInput) HasUpdates() bool {
	return u.TaskStatus != nil ||
		u.Title != nil ||
		u.Description != nil ||
		u.ScheduledAt != nil ||
		u.ClearScheduledAt ||
		u.Color != nil
}

func (t *Task) ApplyUpdate(input *TaskUpdateInput) (*Task, error) {
	if input == nil || !input.HasUpdates() {
		return nil, ErrNoFieldsToUpdate
	}

	newStatus := t.taskStatus
	newTitle := t.title
	newDescription := t.description
	newScheduledAt := t.scheduledAt
	newColor := t.color
	newTargetAt := t.targetAt

	if input.TaskStatus != nil {
		newStatus = *input.TaskStatus
	}

	if input.Title != nil {
		newTitle = *input.Title
	}

	if input.Description != nil {
		newDescription = *input.Description
	}

	if input.ClearScheduledAt {
		newScheduledAt = nil
	} else if input.ScheduledAt != nil {
		newScheduledAt = input.ScheduledAt
	}

	if input.Color != nil {
		newColor = *input.Color
	}

	if t.taskType == TypeScheduled && newScheduledAt != nil {
		newTargetAt = *newScheduledAt
	}

	return NewTask(
		t.id,
		t.userID,
		newTitle,
		t.taskType,
		newStatus,
		newDescription,
		newScheduledAt,
		t.createdAt,
		newTargetAt,
		newColor,
	)
}
