package period

import "errors"

var (
	ErrScheduledTypeNotAllowed = errors.New("scheduled task type is not allowed for period settings")
	ErrInvalidPeriodMinutes    = errors.New("period minutes must be between 1 and 10080")
	ErrInvalidTaskType         = errors.New("invalid task type for period setting")
)
