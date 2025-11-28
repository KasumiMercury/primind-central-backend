package oidcidentity

import (
	"context"
	"errors"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
)

var ErrOIDCIdentityNotFound = errors.New("oidc identity not found")

type OIDCIdentityRepository interface {
	SaveOIDCIdentity(ctx context.Context, identity *OIDCIdentity) error
	GetOIDCIdentityByProviderSubject(ctx context.Context, provider domainoidc.ProviderID, subject string) (*OIDCIdentity, error)
}
