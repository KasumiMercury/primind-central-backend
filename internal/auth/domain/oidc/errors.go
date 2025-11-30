package oidc

import "errors"

var (
	ErrProviderInvalid   = errors.New("provider must be specified")
	ErrStateEmpty        = errors.New("state must be specified")
	ErrNonceEmpty        = errors.New("nonce must be specified")
	ErrCodeVerifierEmpty = errors.New("code verifier must be specified")
	ErrParamsExpired     = errors.New("authentication parameters have expired")
	ErrParamsNotFound    = errors.New("params not found")
)
