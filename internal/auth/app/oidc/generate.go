package oidc

//go:generate mockgen -destination=mock_oidc_provider.go -package=oidc . OIDCProvider,OIDCProviderWithLogin
//go:generate mockgen -destination=mock_session_repository.go -package=oidc github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session SessionRepository
//go:generate mockgen -destination=mock_user_repository.go -package=oidc github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user UserRepository
//go:generate mockgen -destination=mock_oidc_identity_repository.go -package=oidc github.com/KasumiMercury/primind-central-backend/internal/auth/domain/oidcidentity OIDCIdentityRepository
//go:generate mockgen -destination=mock_user_with_identity_repository.go -package=oidc github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc UserWithOIDCIdentityRepository
//go:generate mockgen -destination=mock_session_token_generator.go -package=oidc github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc SessionTokenGenerator
