package repository

import "errors"

var (
	ErrUserRequired          = errors.New("user is required")
	ErrSessionRequired       = errors.New("session is required")
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionAlreadyExpired = errors.New("session already expired")
	ErrOIDCIdentityConflict  = errors.New("oidc identity belongs to a different user")
	ErrIdentityRequired      = errors.New("identity is required")
	ErrParamsRequired        = errors.New("oidc params required")
	ErrParamsAlreadyExpired  = errors.New("oidc params already expired")
)
