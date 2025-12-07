package task

import "errors"

var (
	ErrIDGeneration        = errors.New("failed to generate task ID")
	ErrIDInvalidFormat     = errors.New("task ID must be a valid UUID")
	ErrIDInvalidV7         = errors.New("task ID must be a UUIDv7")
	ErrTaskIDAlreadyExists = errors.New("task ID already exists")

	ErrUserIDEmpty                = errors.New("user ID cannot be empty")
	ErrTitleTooLong               = errors.New("task title cannot exceed 500 characters")
	ErrInvalidTaskType            = errors.New("invalid task type")
	ErrInvalidTaskStatus          = errors.New("invalid task status")
	ErrScheduledAtRequired        = errors.New("scheduledAt is required for tasks with type SCHEDULED")
	ErrScheduledAtNotAllowed      = errors.New("scheduledAt is not allowed for tasks not having type SCHEDULED")
	ErrScheduledAtBeforeCreatedAt = errors.New("scheduledAt cannot be before task creation time")
	ErrTaskNotFound               = errors.New("task not found")
)
