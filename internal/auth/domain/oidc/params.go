package oidc

import (
	"time"

	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
)

type Params struct {
	Provider  oidccfg.ProviderID
	State     string
	Nonce     string
	CreatedAt time.Time
}

type AuthorizationURL string
