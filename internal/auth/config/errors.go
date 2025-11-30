package config

import "errors"

var (
	ErrSessionConfigMissing = errors.New("session config missing")
	ErrOIDCConfigInvalid    = errors.New("oidc config invalid")
)
