package logout

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	domainsession "github.com/KasumiMercury/primind-central-backend/internal/auth/domain/session"
)

var (
	ErrTokenRequired = errors.New("session token is required")
	ErrInvalidToken  = errors.New("invalid session token")
)

type TokenVerifier interface {
	Verify(token string) error
	ExtractSessionID(token string) (string, error)
}

type LogoutRequest struct {
	SessionToken string
}

type LogoutResponse struct {
	Success bool
}

type LogoutUseCase interface {
	Logout(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error)
}

type logoutHandler struct {
	sessionRepo   domainsession.SessionRepository
	tokenVerifier TokenVerifier
	logger        *slog.Logger
}

func NewLogoutHandler(
	sessionRepo domainsession.SessionRepository,
	tokenVerifier TokenVerifier,
) *logoutHandler {
	return &logoutHandler{
		sessionRepo:   sessionRepo,
		tokenVerifier: tokenVerifier,
		logger:        slog.Default().WithGroup("auth").WithGroup("logout"),
	}
}

func (h *logoutHandler) Logout(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error) {
	if req == nil {
		return &LogoutResponse{Success: false}, fmt.Errorf("logout request is nil")
	}

	if req.SessionToken == "" {
		h.logger.Warn("logout called with empty token")

		return &LogoutResponse{Success: false}, ErrTokenRequired
	}

	if err := h.tokenVerifier.Verify(req.SessionToken); err != nil {
		h.logger.Info("session token verification failed", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	rawSessionID, err := h.tokenVerifier.ExtractSessionID(req.SessionToken)
	if err != nil {
		h.logger.Info("session id extraction failed", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	sessionID, err := domainsession.ParseID(rawSessionID)
	if err != nil {
		h.logger.Info("session id in token is invalid", slog.String("error", err.Error()))

		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if err := h.sessionRepo.DeleteSession(ctx, sessionID); err != nil {
		h.logger.Warn("failed to delete session", slog.String("error", err.Error()))

		return nil, fmt.Errorf("failed to logout: %v", err)
	}

	return &LogoutResponse{Success: true}, nil
}
