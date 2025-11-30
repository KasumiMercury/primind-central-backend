package authclient

import "errors"

var (
	ErrUnauthorized           = errors.New("unauthorized: invalid or missing session token")
	ErrAuthServiceUnavailable = errors.New("authentication service unavailable")
)
