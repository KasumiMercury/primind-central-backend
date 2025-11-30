package domain

import "errors"

var (
	ErrUserIDEmpty            = errors.New("user ID must be specified")
	ErrExpiresAtMissing       = errors.New("expiresAt must be specified")
	ErrExpiresBeforeStart     = errors.New("expiresAt must be after createdAt")
	ErrSessionIDEmpty         = errors.New("session ID must be specified")
	ErrSessionIDInvalidFormat = errors.New("session ID must be a valid UUID")
	ErrSessionIDInvalidV7     = errors.New("session ID must be a UUIDv7")
	ErrSessionIDGeneration    = errors.New("failed to generate session ID")
)
