package task

import (
	"fmt"
	"time"
	"unicode/utf8"

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
	TypeUrgent     Type = "urgent"
	TypeNormal     Type = "normal"
	TypeLow        Type = "low"
	TypeHasDueTime Type = "has_due_time"
)

func NewType(t string) (Type, error) {
	switch t {
	case string(TypeUrgent), string(TypeNormal), string(TypeLow), string(TypeHasDueTime):
		return Type(t), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidTaskType, t)
	}
}

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

func NewStatus(s string) (Status, error) {
	switch s {
	case string(StatusActive), string(StatusCompleted):
		return Status(s), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidTaskStatus, s)
	}
}

type Task struct {
	id          ID
	userID      string
	title       string
	taskType    Type
	taskStatus  Status
	description *string
	dueTime     *time.Time
	createdAt   time.Time
}

func NewTask(
	id ID,
	userID string,
	title string,
	taskType Type,
	taskStatus Status,
	description *string,
	dueTime *time.Time,
	createdAt time.Time,
) (*Task, error) {
	normalizedCreatedAt := createdAt.UTC().Truncate(time.Microsecond)

	normalizedDueTime := dueTime
	if dueTime != nil {
		t := dueTime.UTC().Truncate(time.Microsecond)
		normalizedDueTime = &t
	}

	if userID == "" {
		return nil, ErrUserIDEmpty
	}

	if title == "" {
		return nil, ErrTitleEmpty
	}

	if utf8.RuneCountInString(title) > 500 {
		return nil, ErrTitleTooLong
	}

	if taskType == TypeHasDueTime && normalizedDueTime == nil {
		return nil, ErrDueTimeRequired
	}

	return &Task{
		id:          id,
		userID:      userID,
		title:       title,
		taskType:    taskType,
		taskStatus:  taskStatus,
		description: description,
		dueTime:     normalizedDueTime,
		createdAt:   normalizedCreatedAt,
	}, nil
}

func CreateTask(
	userID string,
	title string,
	taskType Type,
	description *string,
	dueTime *time.Time,
) (*Task, error) {
	id, err := NewID()
	if err != nil {
		return nil, err
	}

	return NewTask(
		id,
		userID,
		title,
		taskType,
		StatusActive,
		description,
		dueTime,
		time.Now().UTC(),
	)
}

func (t *Task) ID() ID {
	return t.id
}

func (t *Task) UserID() string {
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

func (t *Task) Description() *string {
	return t.description
}

func (t *Task) DueTime() *time.Time {
	return t.dueTime
}

func (t *Task) CreatedAt() time.Time {
	return t.createdAt
}
