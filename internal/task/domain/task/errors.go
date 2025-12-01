package task

import "errors"

var (
	ErrIDGeneration    = errors.New("failed to generate task ID")
	ErrIDInvalidFormat = errors.New("task ID must be a valid UUID")
	ErrIDInvalidV7     = errors.New("task ID must be a UUIDv7")

	ErrUserIDEmpty       = errors.New("user ID cannot be empty")
	ErrTitleTooLong      = errors.New("task title cannot exceed 500 characters")
	ErrInvalidTaskType   = errors.New("invalid task type")
	ErrInvalidTaskStatus = errors.New("invalid task status")
	ErrDueTimeRequired   = errors.New("due time is required for tasks with type HAS_DUE_TIME")
	ErrTaskNotFound      = errors.New("task not found")
)
