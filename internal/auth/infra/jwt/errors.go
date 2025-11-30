package jwt

import "errors"

var (
	ErrUserRequiredForToken    = errors.New("user is required for session token generation")
	ErrSessionRequiredForToken = errors.New("session is required for token generation")
	ErrUserColorInvalid        = errors.New("user color is invalid")
	ErrJWTSignerCreationFailed = errors.New("jwt signer creation failed")
	ErrSessionIDMissing        = errors.New("session id missing in token")
)
