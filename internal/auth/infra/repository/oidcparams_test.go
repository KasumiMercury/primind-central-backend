package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	domainoidc "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/clock"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/testutil"
)

func TestOIDCParamsRepositoryIntegrationSuccess(t *testing.T) {
	ctx := context.Background()

	client, cleanup := testutil.SetupRedisContainer(ctx, t)
	defer cleanup()

	repo := NewOIDCParamsRepository(client)

	params, err := domainoidc.NewParams(domainoidc.ProviderGoogle, "state-1", "nonce-1", "code-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to create params: %v", err)
	}

	if err := repo.SaveParams(ctx, params); err != nil {
		t.Fatalf("SaveParams returned error: %v", err)
	}

	found, err := repo.GetParamsByState(ctx, "state-1")
	if err != nil {
		t.Fatalf("GetParamsByState returned error: %v", err)
	}

	if found.State() != "state-1" || found.Nonce() != "nonce-1" {
		t.Fatalf("unexpected params data")
	}
}

func TestOIDCParamsRepositoryIntegrationError(t *testing.T) {
	ctx := context.Background()

	client, cleanup := testutil.SetupRedisContainer(ctx, t)
	defer cleanup()

	repo := NewOIDCParamsRepository(client)

	if err := repo.SaveParams(ctx, nil); !errors.Is(err, ErrParamsRequired) {
		t.Fatalf("expected ErrParamsRequired, got %v", err)
	}

	expired, _ := domainoidc.NewParams(domainoidc.ProviderGoogle, "state-x", "nonce-x", "code-x", time.Now().Add(-time.Hour))
	if err := repo.SaveParams(ctx, expired); !errors.Is(err, ErrParamsAlreadyExpired) {
		t.Fatalf("expected ErrParamsAlreadyExpired, got %v", err)
	}

	if _, err := repo.GetParamsByState(ctx, "missing"); !errors.Is(err, domainoidc.ErrParamsNotFound) {
		t.Fatalf("expected ErrParamsNotFound, got %v", err)
	}
}

func TestOIDCParamsRepositoryWithFixedClock(t *testing.T) {
	ctx := context.Background()

	client, cleanup := testutil.SetupRedisContainer(ctx, t)
	defer cleanup()

	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	repo := NewOIDCParamsRepositoryWithClock(client, clock.NewFixedClock(now))

	params, err := domainoidc.NewParams(
		domainoidc.ProviderGoogle,
		"state-1",
		"nonce-1",
		"code-1",
		now.Add(-time.Minute),
	)
	if err != nil {
		t.Fatalf("failed to create params: %v", err)
	}

	if err := repo.SaveParams(ctx, params); err != nil {
		t.Fatalf("SaveParams with fixed clock failed: %v", err)
	}

	found, err := repo.GetParamsByState(ctx, "state-1")
	if err != nil {
		t.Fatalf("GetParamsByState returned error: %v", err)
	}

	if found.State() != "state-1" {
		t.Fatalf("unexpected params data")
	}
}
