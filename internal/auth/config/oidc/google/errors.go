package google

import "errors"

var (
	ErrEnvVarMissing             = errors.New("required environment variable missing")
	ErrGoogleClientSecretMissing = errors.New("google oidc client secret missing")
	ErrGoogleRedirectURIMissing  = errors.New("google oidc redirect uri missing")
	ErrGoogleIssuerInvalid       = errors.New("issuer URL host should contain 'google.com'")
)
