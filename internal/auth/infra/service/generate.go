package auth

//go:generate mockgen -destination=mock_service_oidc.go -package=auth github.com/KasumiMercury/primind-central-backend/internal/auth/app/oidc OIDCParamsGenerator,OIDCLoginUseCase
//go:generate mockgen -destination=mock_service_session.go -package=auth github.com/KasumiMercury/primind-central-backend/internal/auth/app/session ValidateSessionUseCase
//go:generate mockgen -destination=mock_service_logout.go -package=auth github.com/KasumiMercury/primind-central-backend/internal/auth/app/logout LogoutUseCase
