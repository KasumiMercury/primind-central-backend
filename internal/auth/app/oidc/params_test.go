package oidc_test

import (
	"context"
	"errors"
	"testing"

	oidc "github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc"
	domain "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidc"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/infra/repository"
	"github.com/KasumiMercury/primind-central-backend/internal/auth/testutil"
	"go.uber.org/mock/gomock"
)

func TestParamsGeneratorGenerateSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		provider        domain.ProviderID
		expectedURLPart string
	}{
		{
			name:            "generates params for valid provider",
			provider:        domain.ProviderGoogle,
			expectedURLPart: "https://accounts.google.com/o/oauth2/v2/auth",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()

			redisClient, cleanupRedis := testutil.SetupRedisContainer(ctx, t)
			t.Cleanup(cleanupRedis)
			repo := repository.NewOIDCParamsRepository(redisClient)

			mockProvider := oidc.NewMockOIDCProvider(ctrl)
			mockProvider.EXPECT().ProviderID().Return(domain.ProviderGoogle).AnyTimes()
			mockProvider.EXPECT().BuildAuthorizationURL(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(state, nonce, codeChallenge string) string {
					return "https://accounts.google.com/o/oauth2/v2/auth?state=" + state
				})

			providers := map[domain.ProviderID]oidc.OIDCProvider{
				domain.ProviderGoogle: mockProvider,
			}

			generator := oidc.NewParamsGenerator(providers, repo)

			result, err := generator.Generate(ctx, tt.provider)

			if err != nil {
				t.Fatalf("Generate() unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Generate() returned nil result")
			}

			if result.AuthorizationURL == "" {
				t.Error("AuthorizationURL is empty")
			}

			if result.State == "" {
				t.Error("State is empty")
			}

			if len(result.State) != 43 {
				t.Errorf("State length = %d, want 43", len(result.State))
			}
		})
	}
}

func TestParamsGeneratorGenerateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		provider      domain.ProviderID
		setupMocks    func(*gomock.Controller, *testing.T) (map[domain.ProviderID]oidc.OIDCProvider, domain.ParamsRepository)
		expectedErrIs error
	}{
		{
			name:     "provider unsupported",
			provider: domain.ProviderID("unsupported"),
			setupMocks: func(ctrl *gomock.Controller, t *testing.T) (map[domain.ProviderID]oidc.OIDCProvider, domain.ParamsRepository) {
				ctx := context.Background()
				redisClient, cleanupRedis := testutil.SetupRedisContainer(ctx, t)
				t.Cleanup(cleanupRedis)

				return make(map[domain.ProviderID]oidc.OIDCProvider), repository.NewOIDCParamsRepository(redisClient)
			},
			expectedErrIs: oidc.ErrOIDCProviderUnsupported,
		},
		{
			name:     "repository save fails",
			provider: domain.ProviderGoogle,
			setupMocks: func(ctrl *gomock.Controller, _ *testing.T) (map[domain.ProviderID]oidc.OIDCProvider, domain.ParamsRepository) {
				mockRepo := domain.NewMockParamsRepository(ctrl)
				mockRepo.EXPECT().SaveParams(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))

				mockProvider := oidc.NewMockOIDCProvider(ctrl)
				mockProvider.EXPECT().ProviderID().Return(domain.ProviderGoogle).AnyTimes()
				mockProvider.EXPECT().BuildAuthorizationURL(gomock.Any(), gomock.Any(), gomock.Any()).
					Return("https://example.com/auth?state=test")

				providers := map[domain.ProviderID]oidc.OIDCProvider{
					domain.ProviderGoogle: mockProvider,
				}

				return providers, mockRepo
			},
			expectedErrIs: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			providers, repo := tt.setupMocks(ctrl, t)
			ctx := context.Background()

			generator := oidc.NewParamsGenerator(providers, repo)

			result, err := generator.Generate(ctx, tt.provider)

			if err == nil {
				t.Fatalf("Generate() expected error, got result: %+v", result)
			}

			if tt.expectedErrIs != nil && !errors.Is(err, tt.expectedErrIs) {
				t.Errorf("Generate() error = %v, want %v", err, tt.expectedErrIs)
			}

			if result != nil {
				t.Errorf("Generate() result should be nil on error, got: %+v", result)
			}
		})
	}
}
