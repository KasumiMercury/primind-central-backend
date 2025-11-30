package oidcidentity

import (
	"context"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
)

type OIDCIdentityRepository interface {
	SaveOIDCIdentity(ctx context.Context, identity *OIDCIdentity) error
	GetOIDCIdentityByProviderSubject(ctx context.Context, provider domainoidc.ProviderID, subject string) (*OIDCIdentity, error)
}
