package repository

import (
	"context"
	"sync"

	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
)

type inMemoryOIDCParamsRepository struct {
	mu      sync.Mutex
	byState map[string]*domain.Params
}

func NewInMemoryOIDCParamsRepository() domain.ParamsRepository {
	return &inMemoryOIDCParamsRepository{
		mu:      sync.Mutex{},
		byState: make(map[string]*domain.Params),
	}
}

func (r *inMemoryOIDCParamsRepository) SaveParams(_ context.Context, params *domain.Params) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byState[params.State()] = params

	return nil
}

func (r *inMemoryOIDCParamsRepository) GetParamsByState(_ context.Context, state string) (*domain.Params, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	params, ok := r.byState[state]
	if !ok {
		return nil, domain.ErrParamsNotFound
	}

	return params, nil
}
