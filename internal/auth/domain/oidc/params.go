package oidc

import (
	"time"
)

type ProviderID string

const (
	ProviderGoogle ProviderID = "google"
)

type Params struct {
	Provider  ProviderID
	State     string
	Nonce     string
	CreatedAt time.Time
}

type AuthorizationURL string
