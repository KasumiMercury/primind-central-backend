package repository

import (
	"context"
	"sync"
	"time"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity"
)

type IdentityKey struct {
	Provider domainoidc.ProviderID
	Subject  string
}

type IdentityRecord struct {
	identity  *oidcidentity.OIDCIdentity
	createdAt time.Time
}

type inMemoryOIDCIdentityRepository struct {
	mu                sync.Mutex
	byProviderSubject map[IdentityKey]IdentityRecord
}

func NewInMemoryOIDCIdentityRepository() oidcidentity.OIDCIdentityRepository {
	return &inMemoryOIDCIdentityRepository{
		byProviderSubject: make(map[IdentityKey]IdentityRecord),
	}
}

func (r *inMemoryOIDCIdentityRepository) SaveOIDCIdentity(_ context.Context, identity *oidcidentity.OIDCIdentity) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := IdentityKey{
		Provider: identity.Provider(),
		Subject:  identity.Subject(),
	}

	r.byProviderSubject[key] = IdentityRecord{
		identity:  identity,
		createdAt: time.Now(),
	}
	return nil
}

func (r *inMemoryOIDCIdentityRepository) GetOIDCIdentityByProviderSubject(_ context.Context, provider domainoidc.ProviderID, subject string) (*oidcidentity.OIDCIdentity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := IdentityKey{
		Provider: provider,
		Subject:  subject,
	}

	record, exists := r.byProviderSubject[key]
	if !exists {
		return nil, oidcidentity.ErrOIDCIdentityNotFound
	}
	return record.identity, nil
}
