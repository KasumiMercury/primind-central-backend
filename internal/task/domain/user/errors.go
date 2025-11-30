package user

import "errors"

var (
	ErrIDGeneration    = errors.New("failed to generate user ID")
	ErrIDInvalidFormat = errors.New("user ID must be a valid UUID")
	ErrIDInvalidV7     = errors.New("user ID must be a UUIDv7")
)
