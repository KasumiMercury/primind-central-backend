package session

//go:generate mockgen -destination=mock_token_verifier.go -package=session github.com/KasumiMercury/primind-central-backend/internal/auth/app/session TokenVerifier
//go:generate mockgen -destination=mock_session_repository.go -package=session github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session SessionRepository
