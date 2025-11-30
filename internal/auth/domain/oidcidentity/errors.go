package oidcidentity

import "errors"

var (
	ErrUserIDEmpty          = errors.New("user ID must be specified")
	ErrProviderEmpty        = errors.New("provider must be specified")
	ErrSubjectEmpty         = errors.New("subject must be specified")
	ErrOIDCIdentityNotFound = errors.New("oidc identity not found")
)
