package oidcidentity

import (
	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
)

type OIDCIdentity struct {
	userID   user.ID
	provider domainoidc.ProviderID
	subject  string
}

func NewOIDCIdentity(userID user.ID, provider domainoidc.ProviderID, subject string) (*OIDCIdentity, error) {
	if userID == (user.ID{}) {
		return nil, ErrUserIDEmpty
	}

	if provider == "" {
		return nil, ErrProviderEmpty
	}

	if subject == "" {
		return nil, ErrSubjectEmpty
	}

	return &OIDCIdentity{
		userID:   userID,
		provider: provider,
		subject:  subject,
	}, nil
}

func (i *OIDCIdentity) UserID() user.ID {
	return i.userID
}

func (i *OIDCIdentity) Provider() domainoidc.ProviderID {
	return i.provider
}

func (i *OIDCIdentity) Subject() string {
	return i.subject
}
