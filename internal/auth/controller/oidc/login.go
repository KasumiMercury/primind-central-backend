package oidc

import (
	"context"
	"errors"

	oidccfg "github.com/KasumiMercury/primind-central-backend/internal/auth/config/oidc"
)

var (
	ErrInvalidCode  = errors.New("invalid authorization code")
	ErrInvalidState = errors.New("invalid state parameter")
	ErrInvalidNonce = errors.New("nonce validation failed")
)

type OIDCLoginUseCase interface {
	Login(ctx context.Context, req *LoginRequest) (*LoginResult, error)
}

type LoginRequest struct {
	Provider oidccfg.ProviderID
	Code     string
	State    string
}

type LoginResult struct {
	SessionID string
	UserID    string
	CreatedAt int64
	ExpiresAt int64
}
