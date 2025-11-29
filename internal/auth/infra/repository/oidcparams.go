package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/redis/go-redis/v9"
)

var ErrParamsRequired = errors.New("oidc params required")
var ErrParamsAlreadyExpired = errors.New("oidc params already expired")

type paramsRecord struct {
	Provider     string    `json:"provider"`
	State        string    `json:"state"`
	Nonce        string    `json:"nonce"`
	CodeVerifier string    `json:"code_verifier"`
	CreatedAt    time.Time `json:"created_at"`
}

type oidcParamsRepository struct {
	client *redis.Client
}

func NewOIDCParamsRepository(client *redis.Client) domainoidc.ParamsRepository {
	return &oidcParamsRepository{client: client}
}

func (r *oidcParamsRepository) SaveParams(ctx context.Context, params *domainoidc.Params) error {
	if params == nil {
		return ErrParamsRequired
	}

	record := paramsRecord{
		Provider:     string(params.Provider()),
		State:        params.State(),
		Nonce:        params.Nonce(),
		CodeVerifier: params.CodeVerifier(),
		CreatedAt:    params.CreatedAt(),
	}

	ttl := time.Until(params.ExpiresAt())
	if ttl <= 0 {
		return ErrParamsAlreadyExpired
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, r.key(params.State()), payload, ttl).Err()
}

func (r *oidcParamsRepository) GetParamsByState(ctx context.Context, state string) (*domainoidc.Params, error) {
	raw, err := r.client.Get(ctx, r.key(state)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, domainoidc.ErrParamsNotFound
	}

	if err != nil {
		return nil, err
	}

	var record paramsRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}

	return domainoidc.NewParams(domainoidc.ProviderID(record.Provider), record.State, record.Nonce, record.CodeVerifier, record.CreatedAt)
}

func (r *oidcParamsRepository) key(state string) string {
	return fmt.Sprintf("auth:oidc:params:%s", state)
}
