package session

import "errors"

var (
	ErrSessionSecretMissing   = errors.New("session secret is required")
	ErrSessionDurationInvalid = errors.New("session duration must be positive")
)
