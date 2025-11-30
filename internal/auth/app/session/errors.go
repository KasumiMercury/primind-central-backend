package session

import "errors"

var (
	ErrSessionTokenRequired = errors.New("session token is required")
	ErrSessionTokenInvalid  = errors.New("session token is invalid")
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionExpired       = errors.New("session expired")
	ErrRequestNil           = errors.New("request is required")
)
