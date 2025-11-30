package oidc

import "errors"

var (
	ErrClientIDMissing        = errors.New("client ID is required")
	ErrClientSecretMissing    = errors.New("client secret is required")
	ErrRedirectURIMissing     = errors.New("redirect URI is required")
	ErrRedirectSchemeInvalid  = errors.New("redirect URI scheme must be http or https")
	ErrRedirectSchemeMissing  = errors.New("redirect URI must include scheme (http:// or https://)")
	ErrScopesMissing          = errors.New("at least one scope is required")
	ErrScopeOpenIDRequired    = errors.New("'openid' scope is required for OIDC")
	ErrIssuerURLMissing       = errors.New("issuer URL is required")
	ErrIssuerURLSchemeInvalid = errors.New("issuer URL must use https")
	ErrNoOIDCProviders        = errors.New("no oidc providers configured")
	ErrNoProvidersConfigured  = errors.New("no providers configured")
	ErrProviderConfigNil      = errors.New("provider config missing")
	ErrProviderIDMismatch     = errors.New("provider identifier mismatch")
	ErrProviderCoreInvalid    = errors.New("provider core config invalid")
	ErrProviderValidateFail   = errors.New("provider validation failed")
)
