package logout

import "errors"

var (
	ErrSessionTokenRequired = errors.New("session token is required")
	ErrSessionTokenInvalid  = errors.New("session token is invalid")
	ErrRequestNil           = errors.New("request is required")
)
