package oidc

import "errors"

var (
	ErrOIDCNotConfigured       = errors.New("oidc providers are not configured")
	ErrOIDCProviderUnsupported = errors.New("oidc provider is not configured")
	ErrCodeInvalid             = errors.New("authorization code is invalid")
	ErrStateInvalid            = errors.New("state parameter is invalid")
	ErrNonceInvalid            = errors.New("nonce validation failed")
)
