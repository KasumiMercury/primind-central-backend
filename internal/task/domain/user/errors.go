package user

import "errors"

var (
	ErrIDGeneration    = errors.New("failed to generate task ID")
	ErrIDInvalidFormat = errors.New("task ID must be a valid UUID")
	ErrIDInvalidV7     = errors.New("task ID must be a UUIDv7")
)
